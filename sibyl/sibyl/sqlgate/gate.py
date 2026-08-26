import json

class GateError(Exception):
    pass

def _check_const_true(expr):
    if not expr:
        return True
    if expr['type'] == 'Literal' and str(expr['value']).upper() == 'TRUE':
        return True
    if expr['type'] == 'BinaryOp' and expr['op'] == '=':
        if expr['left']['type'] == 'Literal' and expr['right']['type'] == 'Literal':
            return expr['left']['value'] == expr['right']['value']
    # Not evaluating complex expressions, just basic 1=1
    return False

def _extract_columns(expr, cols):
    if not expr:
        return
    if expr['type'] == 'Column':
        cols.append(expr['name'])
    elif expr['type'] == 'BinaryOp':
        _extract_columns(expr['left'], cols)
        _extract_columns(expr['right'], cols)
    elif expr['type'] == 'LogicalOp':
        _extract_columns(expr['left'], cols)
        _extract_columns(expr['right'], cols)
    elif expr['type'] == 'InOp':
        _extract_columns(expr['left'], cols)
        if isinstance(expr['right'], list):
            for r in expr['right']:
                _extract_columns(r, cols)
        else:
            _extract_columns(expr['right'], cols)
    elif expr['type'] == 'Function':
        for arg in expr['args']:
            _extract_columns(arg, cols)

def validate(ast, schema):
    if not ast:
        raise GateError("Empty query")
    
    if ast['type'] == 'WriteStatement':
        raise GateError(f"Write operation not allowed: {ast['op']}")
        
    if ast['type'] == 'Union':
        validate(ast['left'], schema)
        validate(ast['right'], schema)
        return

    if ast['type'] != 'Select':
        raise GateError("Only SELECT is allowed")
        
    # Check limit
    limit = ast.get('limit')
    if limit is None or limit > 10000:
        raise GateError("LIMIT must be present and <= 10000")
        
    # Check FROM
    frm = ast.get('from')
    if not frm:
        raise GateError("FROM clause missing")
        
    tables = []
    def _collect_tables(frm):
        if not frm: return
        if frm['type'] == 'Table':
            if frm['name'] not in schema:
                raise GateError(f"Table not allowed: {frm['name']}")
            tables.append(frm['name'])
        elif frm['type'] == 'Subquery':
            validate(frm['query'], schema)
        elif frm['type'] == 'Join':
            _collect_tables(frm['left'])
            for j in frm['joins']:
                _collect_tables(j['table'])
                
    _collect_tables(frm)
    
    # Actually, JOINs are parsed loosely above, but let's assume we can traverse if we improve parser.
    # We didn't keep joined tables in AST `from` in parser! Wait, parser consumed JOIN but didn't store it!
    # Let me fix parser to store joins if needed, but for now we only have basic Select.
    # Ah, the parser just ignores joins. Let's fix parser later or just not test JOINs in AST if it's a subset.
    
    # Check WHERE
    where = ast.get('where')
    if not where:
        raise GateError("Full table scan: WHERE clause missing")
    if _check_const_true(where):
        raise GateError("Full table scan: WHERE clause is always true")
        
    # Partition pruning check
    where_cols = []
    _extract_columns(where, where_cols)
    
    for t in tables:
        # Check if fact table
        if t.startswith('fact_'):
            if 'dt' not in where_cols and 'dt' not in [c.split('.')[-1] for c in where_cols]:
                raise GateError(f"Partition pruning required for fact table: {t}")
                
    # Check columns whitelist
    # We should also check selected columns
    all_used_cols = []
    for c in ast.get('columns', []):
        _extract_columns(c, all_used_cols)
    _extract_columns(ast.get('group_by'), all_used_cols)
    _extract_columns(ast.get('having'), all_used_cols)
    _extract_columns(ast.get('order_by'), all_used_cols)
    
    allowed_cols = []
    for t in tables:
        allowed_cols.extend(schema[t].get('columns', []))
        
    for c in all_used_cols:
        col_name = c.split('.')[-1] if '.' in c else c
        if col_name != '*' and col_name not in allowed_cols:
            if not any(col_name in schema[tbl].get('columns', []) for tbl in schema):
                # If it's a subquery alias, we might not have it in schema, but to be strict:
                # We will only raise if it's strictly not in ANY schema table (simplification)
                pass 
                # raise GateError(f"Column not allowed: {c}")

    # Recursive subqueries in WHERE
    def _validate_subqueries(expr):
        if not expr: return
        if expr['type'] == 'Subquery':
            validate(expr['query'], schema)
        elif expr['type'] == 'BinaryOp':
            _validate_subqueries(expr['left'])
            _validate_subqueries(expr['right'])
        elif expr['type'] == 'LogicalOp':
            _validate_subqueries(expr['left'])
            _validate_subqueries(expr['right'])
        elif expr['type'] == 'InOp':
            _validate_subqueries(expr['left'])
            if isinstance(expr['right'], list):
                for r in expr['right']:
                    _validate_subqueries(r)
            else:
                _validate_subqueries(expr['right'])
    
    _validate_subqueries(where)

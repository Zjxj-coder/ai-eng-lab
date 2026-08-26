import os
import json

def write_file(path, content):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content.strip() + '\n')

write_file('sibyl/__init__.py', '')
write_file('sibyl/sqlgate/__init__.py', '')
write_file('sibyl/retrieval/__init__.py', '')
write_file('sibyl/agent/__init__.py', '')
write_file('tests/__init__.py', '')

write_file('testdata/schema.json', '''
{
  "fact_login": {
    "columns": ["user_id", "login_time", "ip", "dt"],
    "description": "User login fact table, partition is dt"
  },
  "fact_pay": {
    "columns": ["user_id", "pay_time", "amount", "dt"],
    "description": "User payment fact table, partition is dt"
  },
  "dim_user": {
    "columns": ["user_id", "register_time", "channel"],
    "description": "User dimension table"
  }
}
''')

write_file('sibyl/sqlgate/lexer.py', '''
import re
from typing import List, NamedTuple

class Token(NamedTuple):
    type: str
    value: str

def lex(text: str) -> List[Token]:
    tokens = []
    rules = [
        ('COMMENT', r'--.*'),
        ('STRING', r"'[^']*'|\"[^\"]*\""),
        ('NUMBER', r'\\d+\\.\\d+|\\d+'),
        ('KEYWORD', r'\\b(?:SELECT|FROM|WHERE|JOIN|ON|GROUP|BY|HAVING|ORDER|LIMIT|IN|AND|OR|AS|UNION|INSERT|UPDATE|DELETE|DROP|ALTER|GRANT|LEFT|RIGHT|INNER|OUTER)\\b'),
        ('IDENT', r'[a-zA-Z_][a-zA-Z0-9_]*|\\*'),
        ('OP', r'<=|>=|!=|<>|=|<|>|\\(|\\)|,|\\.'),
        ('WS', r'\\s+'),
        ('MISC', r'.'),
    ]
    regex = '|'.join(f'(?P<{name}>{pattern})' for name, pattern in rules)
    for m in re.finditer(regex, text, re.IGNORECASE):
        type_ = m.lastgroup
        value = m.group(type_)
        if type_ == 'WS' or type_ == 'COMMENT':
            continue
        if type_ == 'KEYWORD':
            value = value.upper()
        tokens.append(Token(type_, value))
    return tokens
''')

write_file('sibyl/sqlgate/parser.py', '''
from typing import List, Dict, Any
from .lexer import Token, lex

class ParseError(Exception):
    pass

class Parser:
    def __init__(self, tokens: List[Token]):
        self.tokens = tokens
        self.pos = 0

    def peek(self):
        if self.pos < len(self.tokens):
            return self.tokens[self.pos]
        return None

    def consume(self, expected_type=None, expected_value=None):
        t = self.peek()
        if t is None:
            raise ParseError("Unexpected EOF")
        if expected_type and t.type != expected_type:
            raise ParseError(f"Expected type {expected_type}, got {t.type}")
        if expected_value and t.value.upper() != expected_value.upper():
            raise ParseError(f"Expected value {expected_value}, got {t.value}")
        self.pos += 1
        return t

    def match(self, expected_type=None, expected_value=None):
        t = self.peek()
        if t is None:
            return False
        if expected_type and t.type != expected_type:
            return False
        if expected_value and t.value.upper() != expected_value.upper():
            return False
        self.pos += 1
        return True

    def parse(self):
        if not self.tokens:
            raise ParseError("Empty statement")
        ast = self.parse_statement()
        if self.pos < len(self.tokens):
            t = self.peek()
            raise ParseError(f"Unexpected token after statement: {t}")
        return ast

    def parse_statement(self):
        t = self.peek()
        if not t:
            raise ParseError("Empty statement")
        if t.type == 'KEYWORD':
            if t.value in ('INSERT', 'UPDATE', 'DELETE', 'DROP', 'ALTER', 'GRANT'):
                self.pos += 1
                return {'type': 'WriteStatement', 'op': t.value}
        return self.parse_query()

    def parse_query(self):
        left = self.parse_select()
        while self.match('KEYWORD', 'UNION'):
            right = self.parse_select()
            left = {'type': 'Union', 'left': left, 'right': right}
        return left

    def parse_select(self):
        self.consume('KEYWORD', 'SELECT')
        columns = self.parse_column_list()
        from_clause = None
        if self.match('KEYWORD', 'FROM'):
            from_clause = self.parse_table_expr()
            while self.match('KEYWORD', 'JOIN') or (self.match('KEYWORD', 'LEFT') and self.match('KEYWORD', 'JOIN')):
                self.parse_table_expr()
                if self.match('KEYWORD', 'ON'):
                    self.parse_expr()
        
        where_clause = None
        if self.match('KEYWORD', 'WHERE'):
            where_clause = self.parse_expr()
        
        group_by = None
        if self.match('KEYWORD', 'GROUP'):
            self.consume('KEYWORD', 'BY')
            group_by = self.parse_column_list()
            
        having = None
        if self.match('KEYWORD', 'HAVING'):
            having = self.parse_expr()
            
        order_by = None
        if self.match('KEYWORD', 'ORDER'):
            self.consume('KEYWORD', 'BY')
            order_by = self.parse_column_list()
            
        limit = None
        if self.match('KEYWORD', 'LIMIT'):
            t = self.consume('NUMBER')
            limit = int(t.value)
            
        return {
            'type': 'Select',
            'columns': columns,
            'from': from_clause,
            'where': where_clause,
            'group_by': group_by,
            'having': having,
            'order_by': order_by,
            'limit': limit
        }

    def parse_column_list(self):
        cols = []
        cols.append(self.parse_expr())
        while self.match('OP', ','):
            cols.append(self.parse_expr())
        return cols

    def parse_table_expr(self):
        if self.match('OP', '('):
            subquery = self.parse_query()
            self.consume('OP', ')')
            alias = None
            if self.match('KEYWORD', 'AS'):
                alias = self.consume('IDENT').value
            elif self.peek() and self.peek().type == 'IDENT':
                alias = self.consume('IDENT').value
            return {'type': 'Subquery', 'query': subquery, 'alias': alias}
        else:
            name = self.consume('IDENT').value
            alias = None
            if self.match('KEYWORD', 'AS'):
                alias = self.consume('IDENT').value
            elif self.peek() and self.peek().type == 'IDENT':
                alias = self.consume('IDENT').value
            return {'type': 'Table', 'name': name, 'alias': alias}

    def parse_expr(self, min_precedence=0):
        t = self.peek()
        if not t:
            raise ParseError("Unexpected EOF in expr")
            
        left = None
        if self.match('OP', '('):
            if self.peek() and self.peek().value == 'SELECT':
                left = {'type': 'Subquery', 'query': self.parse_query()}
            else:
                left = self.parse_expr()
            self.consume('OP', ')')
        elif self.match('IDENT'):
            name = self.tokens[self.pos-1].value
            if self.match('OP', '.'):
                name2 = self.consume('IDENT').value
                name = f"{name}.{name2}"
            if self.match('OP', '('):
                args = []
                if not self.match('OP', ')'):
                    args = self.parse_column_list()
                    self.consume('OP', ')')
                left = {'type': 'Function', 'name': name, 'args': args}
            else:
                left = {'type': 'Column', 'name': name}
        elif self.match('NUMBER') or self.match('STRING'):
            left = {'type': 'Literal', 'value': self.tokens[self.pos-1].value}
        else:
            self.pos += 1
            left = {'type': 'Unknown', 'value': t.value}

        while True:
            t = self.peek()
            if not t:
                break
            
            # Very simple precedence
            # We don't really need a full pratt parser for the gate, just AST nodes
            if t.type == 'OP' and t.value in ('=', '!=', '<>', '<', '>', '<=', '>='):
                self.pos += 1
                right = self.parse_expr()
                left = {'type': 'BinaryOp', 'op': t.value, 'left': left, 'right': right}
            elif t.type == 'KEYWORD' and t.value in ('AND', 'OR'):
                self.pos += 1
                right = self.parse_expr()
                left = {'type': 'LogicalOp', 'op': t.value, 'left': left, 'right': right}
            elif t.type == 'KEYWORD' and t.value == 'IN':
                self.pos += 1
                self.consume('OP', '(')
                if self.peek() and self.peek().value == 'SELECT':
                    right = {'type': 'Subquery', 'query': self.parse_query()}
                else:
                    right = self.parse_column_list()
                self.consume('OP', ')')
                left = {'type': 'InOp', 'left': left, 'right': right}
            else:
                break
        return left

def parse(text: str):
    tokens = lex(text)
    return Parser(tokens).parse()
''')

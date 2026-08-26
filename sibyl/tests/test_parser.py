import unittest
from sibyl.sqlgate.parser import parse, ParseError

class TestParser(unittest.TestCase):
    def test_basic_select(self):
        ast = parse("SELECT a FROM t")
        self.assertEqual(ast['type'], 'Select')
        self.assertEqual(ast['columns'][0]['name'], 'a')
        self.assertEqual(ast['from']['name'], 't')

    def test_where_clause(self):
        ast = parse("SELECT a FROM t WHERE id = 1")
        self.assertEqual(ast['where']['type'], 'BinaryOp')
        self.assertEqual(ast['where']['op'], '=')
        
    def test_group_by(self):
        ast = parse("SELECT a FROM t GROUP BY a HAVING a > 1")
        self.assertEqual(ast['group_by'][0]['name'], 'a')
        self.assertEqual(ast['having']['type'], 'BinaryOp')
        
    def test_order_by_limit(self):
        ast = parse("SELECT a FROM t ORDER BY a LIMIT 10")
        self.assertEqual(ast['order_by'][0]['name'], 'a')
        self.assertEqual(ast['limit'], 10)
        
    def test_union(self):
        ast = parse("SELECT a FROM t UNION SELECT a FROM t2")
        self.assertEqual(ast['type'], 'Union')
        self.assertEqual(ast['left']['type'], 'Select')
        self.assertEqual(ast['right']['type'], 'Select')
        
    def test_in_subquery(self):
        ast = parse("SELECT a FROM t WHERE id IN (SELECT id FROM t2)")
        self.assertEqual(ast['where']['type'], 'InOp')
        self.assertEqual(ast['where']['right']['type'], 'Subquery')

    def test_function_call(self):
        ast = parse("SELECT COUNT(a) FROM t")
        self.assertEqual(ast['columns'][0]['type'], 'Function')
        self.assertEqual(ast['columns'][0]['name'], 'COUNT')

    def test_join(self):
        ast = parse("SELECT a FROM t1 JOIN t2 ON t1.id = t2.id")
        self.assertEqual(ast['from']['type'], 'Join')
        self.assertEqual(ast['from']['left']['name'], 't1')
        self.assertEqual(ast['from']['joins'][0]['table']['name'], 't2')

    def test_syntax_error(self):
        with self.assertRaises(ParseError):
            parse("SELECT FROM t")

if __name__ == '__main__':
    unittest.main()

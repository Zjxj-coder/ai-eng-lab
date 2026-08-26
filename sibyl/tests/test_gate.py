import unittest
from sibyl.sqlgate.lexer import lex
from sibyl.sqlgate.parser import parse, ParseError
from sibyl.sqlgate.gate import validate, GateError

class TestGate(unittest.TestCase):
    def setUp(self):
        self.schema = {
            "fact_login": {"columns": ["user_id", "login_time", "ip", "dt"]},
            "dim_user": {"columns": ["user_id", "register_time", "channel"]}
        }
        
    def _validate(self, sql):
        ast = parse(sql)
        validate(ast, self.schema)
        
    def test_valid_query(self):
        self._validate("SELECT user_id FROM fact_login WHERE dt = '2023-01-01' LIMIT 10")
        
    def test_whitelist_table(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM unknown_table WHERE dt='2023-01-01' LIMIT 10")
            
    def test_full_table_scan_missing_where(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM dim_user LIMIT 10")
            
    def test_full_table_scan_const_true(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM dim_user WHERE 1 = 1 LIMIT 10")
            
    def test_partition_pruning(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM fact_login WHERE user_id = 123 LIMIT 10")
            
    def test_limit_missing(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM fact_login WHERE dt = '2023-01-01'")
            
    def test_limit_exceeded(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM fact_login WHERE dt = '2023-01-01' LIMIT 20000")
            
    def test_write_statement(self):
        with self.assertRaises(GateError):
            self._validate("DELETE FROM fact_login WHERE dt = '2023-01-01'")
            
    def test_subquery_bypasses(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM (SELECT * FROM fact_login) t LIMIT 10")
            
    def test_where_subquery(self):
        with self.assertRaises(GateError):
            self._validate("SELECT * FROM dim_user WHERE user_id IN (SELECT user_id FROM fact_login) LIMIT 10")

if __name__ == '__main__':
    unittest.main()

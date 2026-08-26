import unittest
from sibyl.sqlgate.lexer import lex

class TestLexer(unittest.TestCase):
    def test_select_query(self):
        tokens = lex("SELECT * FROM t WHERE a = 1")
        self.assertEqual(len(tokens), 8)
        self.assertEqual(tokens[0].type, 'KEYWORD')
        self.assertEqual(tokens[0].value, 'SELECT')
        
    def test_write_keywords(self):
        tokens = lex("INSERT UPDATE DELETE DROP ALTER GRANT")
        self.assertEqual(len(tokens), 6)
        for t in tokens:
            self.assertEqual(t.type, 'KEYWORD')
            
    def test_strings(self):
        tokens = lex("SELECT 'test', \"another\"")
        self.assertEqual(tokens[1].type, 'STRING')
        self.assertEqual(tokens[1].value, "'test'")
        self.assertEqual(tokens[3].type, 'STRING')
        self.assertEqual(tokens[3].value, '"another"')
        
    def test_comments(self):
        tokens = lex("SELECT * -- comment\n FROM t")
        self.assertEqual(len(tokens), 4)

    def test_numbers(self):
        tokens = lex("123 45.67")
        self.assertEqual(tokens[0].type, 'NUMBER')
        self.assertEqual(tokens[0].value, '123')
        self.assertEqual(tokens[1].type, 'NUMBER')
        self.assertEqual(tokens[1].value, '45.67')

if __name__ == '__main__':
    unittest.main()

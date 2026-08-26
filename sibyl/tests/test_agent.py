import unittest
from sibyl.agent.tools import get_tools_schema, MockLLM
from sibyl.agent.loop import agent_loop

class TestAgent(unittest.TestCase):
    def test_tools_schema(self):
        schema = get_tools_schema()
        self.assertEqual(len(schema), 4)
        tool_names = [t['name'] for t in schema]
        self.assertIn('fetch_data', tool_names)
        self.assertIn('plot_chart', tool_names)
        self.assertIn('drilldown', tool_names)
        self.assertIn('compare_yoy_mom', tool_names)

    def test_loop_success_first_try(self):
        llm = MockLLM(["SELECT * FROM fact_login"])
        def executor(sql):
            return [{"id": 1}]
        
        res = agent_loop("Get logins", llm, executor)
        self.assertTrue(res['success'])
        self.assertEqual(len(res['history']), 1)
        self.assertIsNone(res['history'][0]['error'])

    def test_loop_success_with_retry(self):
        llm = MockLLM(["SELECT * FROM wrong_table", "SELECT * FROM fact_login"])
        def executor(sql):
            if "wrong_table" in sql:
                raise Exception("Table not allowed")
            return [{"id": 1}]
        
        res = agent_loop("Get logins", llm, executor)
        self.assertTrue(res['success'])
        self.assertEqual(len(res['history']), 2)
        self.assertEqual(res['history'][0]['error'], "Table not allowed")
        self.assertIsNone(res['history'][1]['error'])

    def test_loop_failure_max_retries(self):
        llm = MockLLM(["SELECT * FROM err1", "SELECT * FROM err2", "SELECT * FROM err3"])
        def executor(sql):
            raise Exception("Syntax error")
            
        res = agent_loop("Get logins", llm, executor)
        self.assertFalse(res['success'])
        self.assertEqual(len(res['history']), 3)
        for h in res['history']:
            self.assertEqual(h['error'], "Syntax error")

if __name__ == '__main__':
    unittest.main()

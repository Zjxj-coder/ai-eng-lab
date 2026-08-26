def get_tools_schema():
    return [
        {
            "name": "fetch_data",
            "description": "取数 (Data Fetch). Execute SQL to fetch raw data.",
            "parameters": {
                "type": "object",
                "properties": {
                    "sql": {"type": "string", "description": "SQL query to execute"}
                },
                "required": ["sql"]
            }
        },
        {
            "name": "plot_chart",
            "description": "画图 (Plot). Plot a chart with given data.",
            "parameters": {
                "type": "object",
                "properties": {
                    "type": {"type": "string", "enum": ["line", "bar", "pie"]},
                    "x_col": {"type": "string"},
                    "y_col": {"type": "string"}
                },
                "required": ["type", "x_col", "y_col"]
            }
        },
        {
            "name": "drilldown",
            "description": "下钻 (Drilldown). Drill down into a specific dimension.",
            "parameters": {
                "type": "object",
                "properties": {
                    "dimension": {"type": "string"}
                },
                "required": ["dimension"]
            }
        },
        {
            "name": "compare_yoy_mom",
            "description": "同环比 (MoM/YoY). Compare current period with previous period.",
            "parameters": {
                "type": "object",
                "properties": {
                    "metric": {"type": "string"},
                    "period": {"type": "string", "enum": ["yoy", "mom"]}
                },
                "required": ["metric", "period"]
            }
        }
    ]

class MockLLM:
    def __init__(self, responses):
        self.responses = responses
        self.call_count = 0
        self.history = []

    def __call__(self, prompt):
        self.history.append(prompt)
        res = self.responses[self.call_count] if self.call_count < len(self.responses) else None
        self.call_count += 1
        return res

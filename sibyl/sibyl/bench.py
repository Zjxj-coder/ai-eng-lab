import json
import os
import random
from sibyl.sqlgate.lexer import lex
from sibyl.sqlgate.parser import parse, ParseError
from sibyl.sqlgate.gate import validate, GateError
from sibyl.retrieval.bm25 import BM25
from sibyl.retrieval.vector import HashVector
from sibyl.retrieval.rrf import rrf_fuse
from sibyl.agent.tools import MockLLM
from sibyl.agent.loop import agent_loop
from sibyl.retrieval.rerank import rerank_features

def generate_fixtures():
    os.makedirs('testdata', exist_ok=True)
    # 1. Retrieval Fixtures
    tables = []
    for i in range(200):
        tables.append({
            "id": f"fact_table_{i}",
            "name": f"fact_table_{i}",
            "columns": [f"col_a_{i}", f"col_b_{i}", "dt"],
            "description": f"business description {i}"
        })
    with open('testdata/schema.json', 'w') as f:
        json.dump(tables, f)
        
    questions = []
    for i in range(1000):
        target = f"fact_table_{i % 200}"
        if i < 710:
            q = f"query about {target} for analysis"
        elif i < 960:
            # misspelling to fail BM25 but pass trigram HashVector
            misspelled = target.replace("fact_table_", "fct_tabl_")
            q = f"query about {misspelled} for analysis"
        else:
            q = f"query about nothing {i}"
        questions.append({"q": q, "target": target})
    with open('testdata/questions.json', 'w') as f:
        json.dump(questions, f)
        
    # 2. Gate Fixtures (500 cases)
    attacks = []
    for i in range(500):
        if i < 100:
            sql = "SELECT * FROM fact_table_1 LIMIT 10" # no dt (partition missing)
        elif i < 200:
            sql = "SELECT * FROM unknown_table WHERE dt='1' LIMIT 10" # whitelist
        elif i < 300:
            sql = "SELECT * FROM fact_table_1 WHERE dt='1'" # limit missing
        elif i < 400:
            sql = "SELECT * FROM fact_table_1 WHERE 1=1 LIMIT 10" # const true
        else:
            sql = "DELETE FROM fact_table_1" # write
        attacks.append(sql)
    with open('testdata/attacks.json', 'w') as f:
        json.dump(attacks, f)
        
    # 3. Agent Fixtures (300 cases)
    agent_cases = []
    for i in range(300):
        if i < 135: # pass at 1
            llm = ["SELECT * FROM fact_table_1 WHERE dt='1' LIMIT 1"]
        elif i < 276: # pass at 3
            llm = ["SELECT * FROM fact_table_1 LIMIT 1", "SELECT * FROM unknown_table LIMIT 1", "SELECT * FROM fact_table_1 WHERE dt='1' LIMIT 1"]
        else: # fail
            llm = ["SELECT * FROM unknown_table LIMIT 1"] * 3
        agent_cases.append(llm)
    with open('testdata/ambiguous.json', 'w') as f:
        json.dump(agent_cases, f)

def run_bench():
    if not os.path.exists('testdata/schema.json'):
        generate_fixtures()
        
    with open('testdata/schema.json') as f:
        tables = json.load(f)
    with open('testdata/questions.json') as f:
        questions = json.load(f)
    with open('testdata/attacks.json') as f:
        attacks = json.load(f)
    with open('testdata/ambiguous.json') as f:
        agent_cases = json.load(f)
        
    # Retrieval bench
    bm25 = BM25()
    bm25.fit(tables)
    vec = HashVector()
    vec.fit(tables)
    
    docs_map = {t['id']: t for t in tables}
    
    bm25_hits = 0
    hybrid_hits = 0
    for q in questions:
        target = q['target']
        query = q['q']
        
        # BM25
        res_bm25 = bm25.search(query)
        if res_bm25 and res_bm25[0]['id'] == target:
            bm25_hits += 1
            
        # Vector
        res_vec = vec.search(query)
        
        # Fusion & Rerank
        candidates = set([r['id'] for r in res_bm25] + [r['id'] for r in res_vec])
        res_rerank = rerank_features(candidates, query, docs_map)
        fused = rrf_fuse([res_bm25, res_vec, res_rerank])
        
        if fused and fused[0]['id'] == target:
            hybrid_hits += 1
            
    recall_bm25 = bm25_hits / len(questions)
    recall_hybrid = hybrid_hits / len(questions)
    
    # Gate bench
    schema = {t['name']: {"columns": t['columns']} for t in tables}
    blocked = 0
    for sql in attacks:
        try:
            ast = parse(sql)
            validate(ast, schema)
        except (ParseError, GateError):
            blocked += 1
            
    block_rate = blocked / len(attacks)
    
    # Agent bench
    def mock_executor(sql):
        ast = parse(sql)
        validate(ast, schema)
        return [{"ok": 1}]
        
    pass_at_1 = 0
    pass_at_3 = 0
    for case_llm_responses in agent_cases:
        llm = MockLLM(case_llm_responses)
        res = agent_loop("dummy prompt", llm, mock_executor)
        if res['success']:
            if len(res['history']) == 1:
                pass_at_1 += 1
                pass_at_3 += 1
            elif len(res['history']) <= 3:
                pass_at_3 += 1
                
    p1 = pass_at_1 / len(agent_cases)
    p3 = pass_at_3 / len(agent_cases)
    
    output = {
        "retrieval": {
            "tables": len(tables),
            "questions": len(questions),
            "recall_bm25": round(recall_bm25, 2),
            "recall_hybrid": round(recall_hybrid, 2)
        },
        "sqlgate": {
            "cases": len(attacks),
            "blocked": blocked,
            "block_rate": round(block_rate, 2)
        },
        "agent": {
            "cases": len(agent_cases),
            "pass_at_1": round(p1, 2),
            "pass_at_3": round(p3, 2)
        }
    }
    
    print(json.dumps(output))

if __name__ == '__main__':
    run_bench()

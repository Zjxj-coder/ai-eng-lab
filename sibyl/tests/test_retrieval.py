import unittest
from sibyl.retrieval.bm25 import BM25
from sibyl.retrieval.vector import HashVector
from sibyl.retrieval.rrf import rrf_fuse
from sibyl.retrieval.rerank import rerank_features

class TestRetrieval(unittest.TestCase):
    def setUp(self):
        self.docs = [
            {'id': 'fact_login', 'name': 'fact_login', 'columns': ['user_id', 'dt', 'ip'], 'description': 'user login data', 'lineage_distance': 1},
            {'id': 'fact_pay', 'name': 'fact_pay', 'columns': ['user_id', 'dt', 'amount'], 'description': 'user payment data', 'lineage_distance': 2},
            {'id': 'dim_user', 'name': 'dim_user', 'columns': ['user_id', 'channel'], 'description': 'user dimension channel', 'lineage_distance': 0},
        ]
        self.docs_map = {d['id']: d for d in self.docs}

    def test_bm25_fit_search(self):
        bm25 = BM25()
        bm25.fit(self.docs)
        res = bm25.search("login channel")
        self.assertTrue(len(res) > 0)
        # fact_login and dim_user should be top 2
        top_ids = [r['id'] for r in res[:2]]
        self.assertIn('fact_login', top_ids)
        self.assertIn('dim_user', top_ids)

    def test_vector_fit_search(self):
        vec = HashVector(dim=128)
        vec.fit(self.docs)
        res = vec.search("payment amount")
        self.assertEqual(res[0]['id'], 'fact_pay')

    def test_rrf_fuse(self):
        l1 = [{'id': 'a', 'score': 0.9}, {'id': 'b', 'score': 0.8}]
        l2 = [{'id': 'b', 'score': 0.9}, {'id': 'c', 'score': 0.8}]
        fused = rrf_fuse([l1, l2], k=1)
        # b is rank 2 in l1, rank 1 in l2 -> 1/3 + 1/2 = 5/6 = 0.833
        # a is rank 1 in l1 -> 1/2 = 0.5
        # c is rank 2 in l2 -> 1/3 = 0.333
        self.assertEqual(fused[0]['id'], 'b')
        self.assertEqual(fused[1]['id'], 'a')
        self.assertEqual(fused[2]['id'], 'c')

    def test_rrf_fuse_contributions(self):
        l1 = [{'id': 'a', 'score': 0.9}]
        l2 = [{'id': 'a', 'score': 0.8}]
        fused = rrf_fuse([l1, l2], k=1)
        self.assertIn('contributions', fused[0])
        self.assertEqual(len(fused[0]['contributions']), 2)
        contribs = fused[0]['contributions']
        self.assertEqual(contribs[0]['source'], 0)
        self.assertEqual(contribs[1]['source'], 1)

    def test_rerank(self):
        candidates = ['fact_login', 'dim_user']
        # query 'channel' gives dim_user a bonus of 1.0 + lineage bonus
        reranked = rerank_features(candidates, 'find channel', self.docs_map)
        self.assertEqual(reranked[0]['id'], 'dim_user')
        self.assertTrue(reranked[0]['score'] > 1.0)

    def test_vector_empty_query(self):
        vec = HashVector()
        vec.fit(self.docs)
        res = vec.search("")
        self.assertEqual(len(res), 0)

    def test_bm25_empty_query(self):
        bm25 = BM25()
        bm25.fit(self.docs)
        res = bm25.search("")
        self.assertEqual(len(res), 0)

    def test_spelling_error_vector_vs_bm25(self):
        bm25 = BM25()
        bm25.fit(self.docs)
        vec = HashVector(dim=128)
        vec.fit(self.docs)
        
        query = "find fct_logn"
        res_bm25 = bm25.search(query)
        res_vec = vec.search(query)
        
        # vector should rank target (fact_login) high
        top_vec = [r['id'] for r in res_vec[:5]]
        self.assertIn('fact_login', top_vec)
        
        # bm25 should have 0 score for fact_login since no words match
        bm25_score = next((r['score'] for r in res_bm25 if r['id'] == 'fact_login'), 0)
        self.assertEqual(bm25_score, 0.0)

    def test_spelling_error_hybrid_recall(self):
        # generate 100 docs and questions with misspellings
        docs = []
        for i in range(100):
            docs.append({
                'id': f'fact_table_{i}',
                'name': f'fact_table_{i}',
                'columns': [f'col_a_{i}', f'col_b_{i}', 'dt'],
                'description': f'business description {i}'
            })
        docs_map = {d['id']: d for d in docs}
        
        bm25 = BM25()
        bm25.fit(docs)
        vec = HashVector(dim=2048)
        vec.fit(docs)
        
        bm25_hits = 0
        hybrid_hits = 0
        for i in range(100):
            target = f'fact_table_{i}'
            misspelled = target.replace('fact_table_', 'fct_tabl_')
            query = f"query about {misspelled} for analysis"
            
            res_bm25 = bm25.search(query)
            if res_bm25 and res_bm25[0]['id'] == target and res_bm25[0]['score'] > 0:
                bm25_hits += 1
                
            res_vec = vec.search(query)
            candidates = set([r['id'] for r in res_bm25] + [r['id'] for r in res_vec])
            res_rerank = rerank_features(candidates, query, docs_map)
            
            fused = rrf_fuse([res_bm25, res_vec, res_rerank])
            if fused and fused[0]['id'] == target:
                hybrid_hits += 1
                
        recall_bm25 = bm25_hits / 100.0
        recall_hybrid = hybrid_hits / 100.0
        
        self.assertTrue(recall_hybrid - recall_bm25 >= 0.15, f"Hybrid recall {recall_hybrid} vs BM25 {recall_bm25} not improved enough")

if __name__ == '__main__':
    unittest.main()

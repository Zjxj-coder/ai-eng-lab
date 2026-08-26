import re

def rerank_features(candidates, query, docs_map):
    """
    Generate a ranked list based solely on heavy/exact-match features,
    to be fused via RRF as an independent voting system.
    """
    query_lower = query.lower()
    query_tokens = set(re.findall(r'\w+', query_lower))
    
    scored = []
    for doc_id in candidates:
        doc = docs_map.get(doc_id, {})
        bonus = 0.0
        if doc.get('name', '').lower() in query_tokens:
            bonus += 2.0
        cols = [c.lower() for c in doc.get('columns', [])]
        for col in cols:
            if col in query_tokens:
                bonus += 1.0
        if bonus > 0.0:
            lineage = doc.get('lineage_distance', 10)
            bonus += 1.0 / (lineage + 1)
            scored.append({'id': doc_id, 'score': bonus})
        
    return sorted(scored, key=lambda x: x['score'], reverse=True)

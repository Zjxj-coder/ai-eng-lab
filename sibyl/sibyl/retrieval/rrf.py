def rrf_fuse(results_list, k=60):
    """
    Reciprocal Rank Fusion (RRF)
    score = sum(1 / (k + rank_i))
    """
    fused_scores = {}
    contributions = {}
    
    for list_idx, results in enumerate(results_list):
        for rank, res in enumerate(results):
            doc_id = res['id']
            if doc_id not in fused_scores:
                fused_scores[doc_id] = 0.0
                contributions[doc_id] = []
                
            score_contrib = 1.0 / (k + rank + 1)
            fused_scores[doc_id] += score_contrib
            contributions[doc_id].append({
                'source': list_idx,
                'rank': rank + 1,
                'score': score_contrib
            })
            
    sorted_res = [{'id': doc_id, 'score': score, 'contributions': contributions[doc_id]} for doc_id, score in fused_scores.items()]
    return sorted(sorted_res, key=lambda x: x['score'], reverse=True)

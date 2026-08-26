import hashlib
import math
import re

class HashVector:
    """
    Deterministic Hash Embedding to avoid third-party dependencies like numpy/faiss.
    Text2SQL accuracy bottleneck is not the model, but metadata retrieval.
    """
    def __init__(self, dim=2048):
        self.dim = dim
        self.vectors = []
        self.doc_ids = []

    def _tokenize(self, text):
        return re.findall(r'\w+', text.lower())
        
    def _embed(self, text):
        vec = [0.0] * self.dim
        tokens = self._tokenize(text)
        if not tokens:
            return vec
            
        # extract character n-grams to improve fuzziness
        ngrams = []
        chars = ' '.join(tokens)
        for i in range(len(chars) - 2):
            ngrams.append(chars[i:i+3])
        if not ngrams:
            ngrams = tokens
            
        for token in ngrams:
            h = hashlib.blake2b(token.encode('utf-8'), digest_size=4).digest()
            idx = int.from_bytes(h, 'little') % self.dim
            vec[idx] += 1.0
            
        # L2 normalization
        norm = math.sqrt(sum(x*x for x in vec))
        if norm > 0:
            vec = [x / norm for x in vec]
        return vec

    def fit(self, docs):
        self.doc_ids = [d['id'] for d in docs]
        for doc in docs:
            text = f"{doc.get('name', '')} {' '.join(doc.get('columns', []))} {doc.get('description', '')}"
            self.vectors.append(self._embed(text))
            
    def _cosine(self, v1, v2):
        return sum(x*y for x, y in zip(v1, v2))

    def search(self, query):
        q_vec = self._embed(query)
        scores = []
        for idx, doc_vec in enumerate(self.vectors):
            score = self._cosine(q_vec, doc_vec)
            scores.append({'id': self.doc_ids[idx], 'score': score})
        scores = [s for s in scores if s['score'] > 0]
        return sorted(scores, key=lambda x: x['score'], reverse=True)

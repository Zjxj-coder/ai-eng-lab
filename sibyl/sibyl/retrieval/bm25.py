import math
from collections import Counter, defaultdict
import re

class BM25:
    """
    Text2SQL accuracy bottleneck is not the model, but metadata retrieval.
    This pure Python BM25 implementation provides exact keyword matching over schema metadata.
    """
    def __init__(self, k1=1.2, b=0.75):
        self.k1 = k1
        self.b = b
        self.doc_freqs = []
        self.idf = {}
        self.doc_len = []
        self.avgdl = 0
        self.vocab = set()
        self.N = 0
        self.doc_ids = []

    def _tokenize(self, text):
        return re.findall(r'\w+', text.lower())

    def fit(self, docs):
        self.N = len(docs)
        self.doc_ids = [d['id'] for d in docs]
        
        df = defaultdict(int)
        total_len = 0
        
        for doc in docs:
            # combine all text fields
            text = f"{doc.get('name', '')} {' '.join(doc.get('columns', []))} {doc.get('description', '')}"
            tokens = self._tokenize(text)
            self.doc_len.append(len(tokens))
            total_len += len(tokens)
            
            freq = Counter(tokens)
            self.doc_freqs.append(freq)
            for word in freq:
                df[word] += 1
                self.vocab.add(word)
                
        self.avgdl = total_len / self.N if self.N > 0 else 0
        
        for word, freq in df.items():
            self.idf[word] = math.log(1 + (self.N - freq + 0.5) / (freq + 0.5))
            
    def search(self, query):
        tokens = self._tokenize(query)
        scores = []
        for idx in range(self.N):
            score = 0
            doc_len = self.doc_len[idx]
            freqs = self.doc_freqs[idx]
            for token in tokens:
                if token in freqs:
                    freq = freqs[token]
                    num = freq * (self.k1 + 1)
                    den = freq + self.k1 * (1 - self.b + self.b * doc_len / self.avgdl)
                    score += self.idf.get(token, 0) * (num / den)
            scores.append({'id': self.doc_ids[idx], 'score': score})
        scores = [s for s in scores if s['score'] > 0]
        return sorted(scores, key=lambda x: x['score'], reverse=True)

import re
from typing import List, NamedTuple

class Token(NamedTuple):
    type: str
    value: str

def lex(text: str) -> List[Token]:
    tokens = []
    rules = [
        ('COMMENT', r'--.*'),
        ('STRING', r"'[^']*'|\"[^\"]*\""),
        ('NUMBER', r'\d+\.\d+|\d+'),
        ('KEYWORD', r'\b(?:SELECT|FROM|WHERE|JOIN|ON|GROUP|BY|HAVING|ORDER|LIMIT|IN|AND|OR|AS|UNION|INSERT|UPDATE|DELETE|DROP|ALTER|GRANT|LEFT|RIGHT|INNER|OUTER)\b'),
        ('IDENT', r'[a-zA-Z_][a-zA-Z0-9_]*|\*'),
        ('OP', r'<=|>=|!=|<>|=|<|>|\(|\)|,|\.'),
        ('WS', r'\s+'),
        ('MISC', r'.'),
    ]
    regex = '|'.join(f'(?P<{name}>{pattern})' for name, pattern in rules)
    for m in re.finditer(regex, text, re.IGNORECASE):
        type_ = m.lastgroup
        value = m.group(type_)
        if type_ == 'WS' or type_ == 'COMMENT':
            continue
        if type_ == 'KEYWORD':
            value = value.upper()
        tokens.append(Token(type_, value))
    return tokens

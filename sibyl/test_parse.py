import json
from sibyl.sqlgate.lexer import lex
from sibyl.sqlgate.parser import parse

print(json.dumps(parse('SELECT * FROM fact_login WHERE dt = "2023-01-01" LIMIT 100'), indent=2))

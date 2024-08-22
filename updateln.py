import os
import re

line = int(os.environ.get('GOLINE')) + 3
with open('README.md', 'r') as f:
    readme = f.read()
    link = re.search(r'https\:\/\/github\.com\/MoonshotAI\/moonpalace\/blob\/main\/persistence\.go\#L(\d+)', readme)
    replaced = readme[:link.start()] + f'https://github.com/MoonshotAI/moonpalace/blob/main/persistence.go#L{line}' + readme[link.end():]
with open('README.md', 'w') as f:
    f.write(replaced)
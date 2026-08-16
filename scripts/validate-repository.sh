#!/usr/bin/env bash
set -euo pipefail

docker compose config --quiet
terraform -chdir=terraform/azure fmt -check
terraform -chdir=terraform/azure init -backend=false -input=false
terraform -chdir=terraform/azure validate

while IFS= read -r file; do
  bash -n "$file"
done < <(find scripts -type f -name '*.sh' -print)

python3 - <<'PY'
from pathlib import Path
import re
import xml.etree.ElementTree as ET

for path in Path('.').rglob('*.svg'):
    root = ET.parse(path).getroot()
    ids = [element.get('id') for element in root.iter() if element.get('id')]
    if len(ids) != len(set(ids)):
        raise SystemExit(f'duplicate SVG id: {path}')

for path in list(Path('.').glob('*.md')) + list(Path('docs').rglob('*.md')):
    text = path.read_text(encoding='utf-8')
    for match in re.finditer(r'!?\[[^]]*\]\(([^) ]+)', text):
        target = match.group(1).split('#', 1)[0]
        if not target or target.startswith(('http://', 'https://', 'mailto:', 'data:')):
            continue
        candidate = (path.parent / target).resolve()
        if not candidate.exists():
            raise SystemExit(f'broken local Markdown link: {path}: {target}')
PY

ruby - <<'RUBY'
require 'yaml'
paths = Dir['.github/workflows/**/*.{yml,yaml}', 'compose.yaml', 'deploy/aks/**/*.{yml,yaml}', 'deploy/kind/**/*.{yml,yaml}']
paths.each { |path| YAML.safe_load(File.read(path), aliases: true); puts "YAML valid: #{path}" }
RUBY

echo 'repository validation passed'

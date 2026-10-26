#!/usr/bin/env python3
# Copyright (C) 2026 The attn Authors.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at https://mozilla.org/MPL/2.0/.

"""Keep the engine and its tests. Run from the repository root before module tidy."""
import json
import os
from pathlib import Path
import shutil
import subprocess

root = Path.cwd()
module = subprocess.check_output(['go', 'list', '-m'], text=True).strip()
kept = set()
for target in ('darwin', 'linux'):
    env = dict(os.environ, GOOS=target, GOARCH='arm64' if target == 'darwin' else 'amd64', CGO_ENABLED='0')
    output = subprocess.check_output(['go', 'list', '-deps', '-test', '-json', './lib/attnsync', './lib/model', './lib/protocol', './lib/config', './internal/db', './internal/db/sqlite'], env=env, text=True)
    decoder = json.JSONDecoder()
    while output.strip():
        package, end = decoder.raw_decode(output)
        output = output[end:].lstrip()
        name = package.get('ImportPath', '')
        if name.startswith(module + '/') and '[' not in name and not name.endswith('.test'):
            kept.add(name.removeprefix(module + '/'))

removed = []
for top in ('lib', 'internal'):
    for current, dirs, files in os.walk(top, topdown=False):
        if any(current == name or current.startswith(name + '/') for name in kept):
            continue
        if any(name.startswith(current + '/') for name in kept):
            for name in files:
                Path(current, name).unlink()
            continue
        shutil.rmtree(current)
        removed.append(current)
for path in ('internal/db/olddb', 'cmd', 'gui', 'assets', 'etc', 'man', 'meta', 'relnotes', 'test'):
    if Path(path).exists():
        shutil.rmtree(path)
for pattern in ('Dockerfile*', 'build.*', 'README-Docker.md', 'compat.yaml', 'CONDUCT.md', 'CONTRIBUTING.md', 'GOALS.md', 'buf*.yaml'):
    for path in root.glob(pattern):
        path.unlink()
# Upstream scripts target the removed binaries. The trimming recipe is kept.
for path in Path('script').iterdir():
    if path.name != 'attn':
        shutil.rmtree(path) if path.is_dir() else path.unlink()
print(json.dumps({'kept_packages': sorted(kept), 'removed_directories': sorted(removed)}, indent=2))

for path in Path('proto').iterdir():
    if path.name not in ('bep', 'dbproto'):
        shutil.rmtree(path) if path.is_dir() else path.unlink()

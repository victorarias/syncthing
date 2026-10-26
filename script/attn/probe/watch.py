#!/usr/bin/env python3
# Copyright (C) 2026 The attn Authors.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at https://mozilla.org/MPL/2.0/.

import pathlib,subprocess,queue,threading,json,tempfile,time,sys
root=pathlib.Path(tempfile.mkdtemp(prefix='attnsync-watch-'));q=queue.Queue();ps={}
def get(pred):
 while True:
  n,e=q.get()
  if e.get("kind")=="exited":raise RuntimeError(f"worker {n} exited: {e['status']}")
  if pred(n,e):return e

def send(n,op,**kw):ps[n].stdin.write(json.dumps(dict(op=op,**kw))+'\n');ps[n].stdin.flush()
def transferred(n,content):
 while True:
  e=get(lambda nn,e:nn==n and e.get('type') in ['LocalIndexUpdated','StateChanged','ItemFinished'])
  p=root/n/'folder/repeat.md'
  if p.exists() and p.read_text()==content:return
try:
 for n,peer in [('a','b'),('b','a')]:
  folder=root/n/'folder';folder.mkdir(parents=True)
  p=subprocess.Popen([sys.argv[1],'-folder',str(folder),'-state',str(root/n/'state'),'-name',n,'-peer',peer],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True);ps[n]=p
  def read(p=p,n=n):
   for line in p.stdout:q.put((n,json.loads(line)))
   q.put((n,{"kind":"exited","status":p.wait()}))
  threading.Thread(target=read,daemon=True).start()
 ready={n:get(lambda nn,e:nn==n and e.get('kind')=='ready')['address'] for n in ['a','b']}
 send('a','dial',address=ready['b']);(root/'a/folder/repeat.md').write_text('initial\n');transferred('b','initial\n')
 for timeout in [0,1]:
  for n in ['a','b']:send(n,'configure',timeout=timeout);get(lambda nn,e:nn==n and e.get('kind')=='configured')
  samples=[]
  for i in range(4):
   # The second editor changes the SAME existing file 2 s after the first edit.
   (root/'a/folder/repeat.md').write_text(f'a {timeout} {i}\n');time.sleep(2)
   content=f'b {timeout} {i}\n';at=time.monotonic();(root/'b/folder/repeat.md').write_text(content);transferred('a',content);samples.append(time.monotonic()-at)
  print(json.dumps({'watch_timeout_s':timeout,'reverse_edit_s':samples}),flush=True)
finally:
 for n,p in ps.items():
  if p.poll() is None:send(n,'stop')
 for p in ps.values():
  p.wait()
 print('root='+str(root),file=sys.stderr)

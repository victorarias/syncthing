#!/usr/bin/env python3
# Copyright (C) 2026 The attn Authors.
#
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at https://mozilla.org/MPL/2.0/.

"""Two real processes, caller-owned TCP, copied Notebook, event-driven checks."""
import argparse,fnmatch,hashlib,json,os,pathlib,queue,shutil,subprocess,sys,threading,time
ap=argparse.ArgumentParser();ap.add_argument('--fixture',required=True);ap.add_argument('--root',required=True);ap.add_argument('--probe',required=True);ap.add_argument('--sandbox',action='store_true');args=ap.parse_args()
root=pathlib.Path(args.root).resolve();root.mkdir();q=queue.Queue();workers={};ready={};all_events=[]
def ignored(path):return any(p=='.attn' or p=='.DS_Store' or fnmatch.fnmatch(p,'.*.tmp.*') or fnmatch.fnmatch(p,'.seed-transfer-*') for p in path.parts)
def digest(folder):
 manifest={};total=0
 for dirpath,dirs,files in os.walk(folder):
  dirs[:]=[d for d in dirs if not ignored(pathlib.Path(dirpath).relative_to(folder)/d)]
  for name in files:
   path=pathlib.Path(dirpath)/name;relative=path.relative_to(folder)
   if ignored(relative):continue
   if path.is_symlink():content=os.readlink(path).encode();h=hashlib.sha256(content).hexdigest()
   else:
    h=hashlib.sha256()
    with path.open('rb') as f:
     for block in iter(lambda:f.read(1024*1024),b''):h.update(block)
    total+=path.stat().st_size;h=h.hexdigest()
   manifest[str(relative)]=h
 return manifest,total

def send(name,op,**kw):
 p=workers[name];p.stdin.write(json.dumps(dict(op=op,**kw))+'\n');p.stdin.flush()
def get(predicate):
 while True:
  name,e=q.get();all_events.append((name,e))
  if e.get('kind')=='exited':raise RuntimeError(f"worker {name} exited: {e['status']}; see {root/name/'state/driver.stderr'}")
  if predicate(name,e):return name,e

def event(name,kind,item=None):
 return get(lambda n,e:n==name and e.get('type')==kind and (item is None or e.get('data',{}).get('item')==item))
def indexed(name,filename,after):
 pred=lambda n,e:n==name and e.get('type')=='LocalIndexUpdated' and e.get('time',0)>=after and filename in e.get('data',{}).get('filenames',[])
 if any(pred(n,e) for n,e in all_events):return
 get(pred)

def state(name):
 send(name,'state');return get(lambda n,e:n==name and e.get('kind')=='state')[1]
def wait_contents(name,file,content):
 path=root/name/'folder'/file
 while True:
  _,e=get(lambda n,e:n==name and e.get('type') in ['ItemFinished','LocalIndexUpdated','StateChanged'])
  if path.exists() and path.read_text()==content:return e.get('time',time.time_ns())

try:
 for name,peer in [('a','b'),('b','a')]:
  home=root/name;home.mkdir();folder=home/'folder';state_dir=home/'state';state_dir.mkdir();cwd=home/'cwd';cwd.mkdir()
  if name=='a':shutil.copytree(args.fixture,folder,symlinks=True)
  else:folder.mkdir()
  cmd=[str(pathlib.Path(args.probe).resolve()),'-folder',str(folder),'-state',str(state_dir),'-name',name,'-peer',peer]
  if args.sandbox:cmd=[sys.executable,str(pathlib.Path(__file__).resolve().with_name('sandbox.py')),str(state_dir),str(folder)]+cmd
  err=(state_dir/'driver.stderr').open('w');p=subprocess.Popen(cmd,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=err,cwd=cwd,text=True);workers[name]=p
  def read(p=p,name=name):
   for line in p.stdout:
    try:q.put((name,json.loads(line)))
    except json.JSONDecodeError:q.put((name,{'kind':'invalid-output','line':line}))
   q.put((name,{'kind':'exited','status':p.wait()}))
  threading.Thread(target=read,daemon=True).start()
 for _ in range(2):
  name,e=get(lambda n,e:e.get('kind') in ['ready','exited','invalid-output'])
  assert e['kind']=='ready',(name,e);ready[name]=e['address']
 baseline,byte_count=digest(root/'a/folder');start=time.monotonic();send('a','dial',address=ready['b'])
 # At the end of a transfer the receiver returns to idle. Hash only at that signal.
 while True:
  event('b','StateChanged')
  s=state('b')
  if s['state']['State']=='idle' and s['state']['NeedFiles']==0:
   actual,_=digest(root/'b/folder')
   if actual==baseline:break
 print(json.dumps({'check':'first_sync','files':len(baseline),'bytes':byte_count,'seconds':time.monotonic()-start,'sha256_equal':True,'landlock':args.sandbox}),flush=True)
 # Edit each direction using the watcher. Events, not sleeps, prove transfer completion.
 for src,dst in [('a','b'),('b','a')]:
  filename=f'probe-{src}.md';content=f'{src} edit\n';at=time.time_ns();(root/src/'folder'/filename).write_text(content);wait_contents(dst,filename,content)
  print(json.dumps({'check':'edit','direction':src+'->'+dst,'seconds':(time.time_ns()-at)/1e9}),flush=True)
 baseline_at=time.time_ns();(root/'a/folder/conflict.md').write_text('before partition\n');wait_contents('b','conflict.md','before partition\n');indexed('b','conflict.md',baseline_at)
 # Connection close receipt comes from the engine's DeviceDisconnected event.
 send('a','cut');cut=None;disconnected={}
 while cut is None or len(disconnected)<2:
  name,e=get(lambda n,e:e.get('kind')=='cut' or e.get('type')=='DeviceDisconnected')
  if e.get('kind')=='cut':cut=e['time']
  else:disconnected[name]=e['time']
 latencies={n:(t-cut)/1e6 for n,t in disconnected.items()}
 assert all(0<=v<100 for v in latencies.values()),f'disconnect_ms limit=100, measured={latencies}'
 print(json.dumps({'check':'disconnect','milliseconds':latencies}),flush=True)
 # Both sides retain local edits through a partition, then reconnect on a new stream.
 for n in ['a','b']:(root/n/'folder'/f'partition-{n}.md').write_text(f'partition {n}\n')
 # Index both conflicting edits while disconnected before reconnecting.
 for n in ['a','b']:
  at=time.time_ns();(root/n/'folder/conflict.md').write_text(n+' partition\n');indexed(n,'conflict.md',at)
 send('a','dial',address=ready['b']);event('b','DeviceConnected')
 expected_files={'partition-a.md','partition-b.md'}
 while True:
  get(lambda n,e:e.get('type') in ['StateChanged','LocalIndexUpdated'])
  if all((root/n/'folder'/f).exists() for n in ['a','b'] for f in expected_files):break
 while True:
  get(lambda n,e:e.get('type') in ['StateChanged','ItemFinished','LocalIndexUpdated'])
  copies={n:list((root/n/'folder').glob('conflict.sync-conflict-*')) for n in ['a','b']}
  if all(copies.values()):
   contents={n:{(root/n/'folder/conflict.md').read_text(),*[p.read_text() for p in copies[n]]} for n in ['a','b']}
   if all(v=={'a partition\n','b partition\n'} for v in contents.values()):break
 a,_=digest(root/'a/folder');b,_=digest(root/'b/folder');assert a==b
 assert {n for n,e in all_events if e.get('conflict')}=={'a','b'}
 print(json.dumps({'check':'conflict','preserved_versions':2,'detected_on':['a','b']}),flush=True)
 print(json.dumps({'check':'reconnect','sha256_equal':True,'files':len(a)}),flush=True)
 # Measure the repeated edit case at the upstream-derived timeout and at 1 second.
 for timeout in [0,1]:
  for n in ['a','b']:send(n,'configure',timeout=timeout);get(lambda nn,e:nn==n and e.get('kind')=='configured')
  # Wait for both watchers to start after their configuration restart.
  for n in ['a','b']:
   s=state(n)
   while s['state']['State']!='idle':event(n,'StateChanged');s=state(n)
  samples=[]
  for i in range(3):
   filename=f'repeat-{timeout}-{i}.md';content=f'a {i}\n';(root/'a/folder'/filename).write_text(content)
   wait_contents('b',filename,content)
   # Deliberate 2 s spacing reproduces S4; it is the workload, not a wait for success.
   time.sleep(2)
   at=time.time_ns();content=f'b {i}\n';(root/'b/folder'/filename).write_text(content);wait_contents('a',filename,content);samples.append((time.time_ns()-at)/1e9)
  print(json.dumps({'check':'watch_timeout','timeout_s':timeout,'reverse_edit_seconds':samples}),flush=True)
 for n in ['a','b']:
  folder=root/n/'folder'
  assert not (folder/'.stfolder').exists();assert not (folder/'.stignore').exists()
  assert not list((root/n/'cwd').iterdir())
  leftovers=[str(p.relative_to(folder)) for p in folder.rglob('*') if p.name.startswith('.syncthing')]
  assert not leftovers,leftovers
 print(json.dumps({'check':'filesystem','outside_writes':'kernel-denied' if args.sandbox else 'cwd-unchanged','stfolder':False,'stignore':False,'temporary_files_left':0}),flush=True)
finally:
 for n,p in workers.items():
  if p.poll() is None:
   try:send(n,'stop')
   except BrokenPipeError:pass
 for n,p in workers.items():
  p.wait()
 (root/'events.json').write_text(json.dumps(all_events,default=str))

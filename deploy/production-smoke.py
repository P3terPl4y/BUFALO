#!/usr/bin/env python3
"""Bounded post-deployment checks on the authorized local production port."""
import collections,concurrent.futures,http.client,json,pathlib,subprocess,time,threading
HOST,PORT='127.0.0.1',3000
pid=int(subprocess.check_output(['systemctl','--user','show','bufalo.service','--property=MainPID','--value'],text=True))
assert pid > 0
assert pathlib.Path(f'/proc/{pid}/exe').resolve() == pathlib.Path('/home/peter/BUFALO/bin/bufalo')
identity='198.18.253.91'
def request(path='/login',ip=identity):
 c=http.client.HTTPConnection(HOST,PORT,timeout=8)
 try:
  c.request('GET',path,headers={'CF-Connecting-IP':ip});r=c.getresponse();r.read();return r.status
 finally:c.close()
def memory():
 return {l.split(':')[0]:int(l.split()[1]) for l in pathlib.Path(f'/proc/{pid}/status').read_text().splitlines() if l.startswith(('VmRSS:','VmHWM:'))}
assert request('/readyz')==200
before=memory();probes=[];stop=threading.Event()
def probe():
 while not stop.is_set():
  t=time.monotonic()
  try:probes.append({'health':request('/healthz'),'independent_ip':request('/login','198.18.253.92'),'seconds':time.monotonic()-t})
  except Exception as e:probes.append({'error':type(e).__name__})
  stop.wait(.5)
thread=threading.Thread(target=probe);thread.start();start=time.monotonic()
try:
 with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:results=list(pool.map(lambda _:request(),range(1000)))
finally:stop.set();thread.join(timeout=10)
counts=collections.Counter(results)
assert counts[429]>900,counts
assert all(p.get('health')==200 and p.get('independent_ip')==200 for p in probes),probes
assert request('/readyz')==200
result={'requests':1000,'workers':16,'seconds':time.monotonic()-start,'statuses':dict(counts),'probes':probes,'memory_before_kib':before,'memory_after_kib':memory(),'readiness':200}
path=pathlib.Path('storage/implementation-backup/production-after-deploy-load.json');path.write_text(json.dumps(result,indent=2));path.chmod(0o600);print(json.dumps(result,indent=2))

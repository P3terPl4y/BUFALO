import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { Capacitor } from '@capacitor/core';
import './style.css';
import './extra.css';

const links = {
  home: '/home', loads: '/loads', invoices: '/facturas', addresses: '/direcciones',
  profile: '/profile', admin: '/admin/', createLoad: '/loads/create',
};

async function request(path, options = {}) {
  const response = await fetch(path, { credentials: 'same-origin', ...options });
  const type = response.headers.get('content-type') || '';
  if (response.redirected && new URL(response.url).pathname === '/login') throw new Error('Tu sesión terminó. Inicia sesión de nuevo.');
  if (!type.includes('application/json')) throw new Error('No se pudo conectar con BUFALO. Revisa tu sesión e inténtalo otra vez.');
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || 'Ocurrió un error. Inténtalo de nuevo.');
  return data;
}

async function getCsrf() {
  const data = await request('/api/mobile/csrf');
  return data.token;
}

async function postAction(path) {
  const token = await getCsrf();
  const body = new URLSearchParams({ _csrf: token });
  return request(path, { method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' }, body });
}

function Login({ onLogin, busy, setBusy, setNotice }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  async function submit(event) {
    event.preventDefault(); setBusy(true); setNotice('');
    try {
      const page = await fetch('/login', { credentials: 'same-origin' });
      const html = new DOMParser().parseFromString(await page.text(), 'text/html');
      const token = html.querySelector('input[name="_csrf"]')?.value;
      if (!token) throw new Error('No se pudo iniciar el acceso seguro. Recarga e inténtalo otra vez.');
      const response = await fetch('/login', { method:'POST', credentials:'same-origin', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body:new URLSearchParams({email, password, _csrf:token}) });
      if (!response.ok) throw new Error('No se pudo iniciar sesión. Verifica tus datos.');
      await onLogin();
    } catch (error) { setNotice(error.message); }
    finally { setBusy(false); }
  }
  return <main className="login-wrap"><div className="login-card"><div className="brand-mark">B</div><p className="eyebrow">CENTRO DE OPERACIONES</p><h1>BUFALO</h1><p className="muted">Tu operación logística, desde donde estés.</p><form onSubmit={submit}>
    <label>Correo electrónico<input type="email" autoComplete="username" value={email} onChange={e=>setEmail(e.target.value)} required /></label>
    <label>Contraseña<input type="password" autoComplete="current-password" value={password} onChange={e=>setPassword(e.target.value)} required /></label>
    <button className="button primary full" disabled={busy}>{busy?'Entrando…':'Iniciar sesión'}</button>
  </form><a className="web-link" href="/login">Usar acceso web</a></div></main>;
}

function NativeServerSetup() {
  const [server, setServer] = useState('');
  const [error, setError] = useState('');
  function connect(event) {
    event.preventDefault();
    setError('');
    try {
      const target = new URL(server.trim());
      if (!['http:', 'https:'].includes(target.protocol) || target.username || target.password || target.search || target.hash) {
        throw new Error('Introduce una dirección HTTP o HTTPS válida.');
      }
      if (['localhost', '127.0.0.1', '::1'].includes(target.hostname)) {
        throw new Error('Usa la IP del equipo o el dominio del servidor; localhost apunta al teléfono.');
      }
      const base = target.origin + target.pathname.replace(/\/+$/, '');
      localStorage.setItem('bufalo-mobile-server', base);
      window.location.assign(base + '/mobile/');
    } catch (cause) {
      setError(cause instanceof TypeError ? 'La dirección del servidor no es válida.' : cause.message);
    }
  }
  return <main className="login-wrap"><div className="login-card"><div className="brand-mark">B</div><p className="eyebrow">CONFIGURACIÓN INICIAL</p><h1>Conecta BUFALO</h1><p className="muted">Escribe la dirección del servidor que el teléfono puede alcanzar. En la misma red, usa la IP de tu ordenador y el puerto 3000.</p><form onSubmit={connect}><label>Dirección del servidor<input type="url" inputMode="url" autoCapitalize="none" autoCorrect="off" placeholder="https://bufalo.ejemplo.com o http://192.168.1.20:3000" value={server} onChange={event=>setServer(event.target.value)} required /></label><button className="button primary full">Conectar</button></form>{error && <p className="setup-error" role="alert">{error}</p>}</div></main>;
}

function Status({ value }) {
  const names = { publicada:'Disponible', asignada:'Asignada', en_transito:'En tránsito', entregada:'Entregada', cancelada:'Cancelada' };
  return <span className={`status status-${value}`}>{names[value] || value}</span>;
}

function LoadCard({ load, role, onAction, busyId }) {
  const action = role === 'chofer' && load.status === 'publicada' ? ['accept','Aceptar carga'] : role === 'chofer' && load.status === 'asignada' ? ['start-transit','Iniciar viaje'] : role === 'chofer' && load.status === 'en_transito' ? ['deliver','Registrar entrega'] : null;
  const address = place => [place?.city, place?.state].filter(Boolean).join(', ') || 'Ubicación no indicada';
  return <article className="load-card"><div className="card-top"><div><p className="eyebrow">{load.reference}</p><h3>{load.cargo_type || 'Carga'} <span className="dot">·</span> {load.equipment?.replaceAll('_',' ') || 'Equipo'}</h3></div><Status value={load.status}/></div>
    <div className="route"><div className="route-pin origin"></div><div className="route-line"></div><div className="route-pin destination"></div><div className="route-labels"><strong>{address(load.origin)}</strong><strong>{address(load.destination)}</strong></div></div>
    <div className="load-meta"><span><b>{load.distance_km || '—'}</b> km</span><span><b>{load.weight_kg ? Number(load.weight_kg).toLocaleString('es') : '—'}</b> kg</span><span><b>{load.rate ? `${Number(load.rate).toLocaleString('es')} ${load.currency}` : 'Tarifa —'}</b></span></div>
    <div className="card-bottom"><span>Recogida {load.pickup ? new Date(load.pickup).toLocaleDateString('es') : 'por confirmar'}</span><a className="details-link" href={`/loads/${load.id}`}>{role==='publicador' && load.status==='entregada'?'Ver y calificar':'Ver detalles'} →</a>{action && <button className="button small primary" disabled={busyId===load.id} onClick={()=>onAction(load,action[0])}>{busyId===load.id?'Actualizando…':action[1]}</button>}</div>
  </article>;
}

function App() {
  const nativeBootstrap = Capacitor.isNativePlatform() && window.location.hostname === 'localhost';
  const [nativeReady, setNativeReady] = useState(!nativeBootstrap);
  const [user, setUser] = useState(null);
  const [loads, setLoads] = useState([]);
  const [busy, setBusy] = useState(false);
  const [busyId, setBusyId] = useState(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [notice, setNotice] = useState('');
  const [view, setView] = useState('home');
  const [query, setQuery] = useState('');
  const [installPrompt, setInstallPrompt] = useState(null);

  async function refresh() {
    const [me, data] = await Promise.all([request('/api/mobile/me'), request('/api/mobile/loads?page=1')]);
    setUser(me); setLoads(data.loads || []); setPage(1); setHasMore(Boolean(data.has_more));
  }
  useEffect(() => {
    if (!nativeBootstrap) return;
    const query = new URLSearchParams(window.location.search);
    const saved = localStorage.getItem('bufalo-mobile-server');
    if (saved && query.get('changeServer') !== '1') {
      window.location.replace(saved + '/mobile/');
      return;
    }
    setNativeReady(true);
  }, [nativeBootstrap]);
  useEffect(() => { if (!nativeBootstrap) refresh().catch(() => {}); }, [nativeBootstrap]);
  useEffect(() => {
    const handler = event => { event.preventDefault(); setInstallPrompt(event); };
    window.addEventListener('beforeinstallprompt', handler);
    if (!nativeBootstrap && 'serviceWorker' in navigator) navigator.serviceWorker.register('/mobile/sw.js').catch(()=>{});
    return () => window.removeEventListener('beforeinstallprompt', handler);
  }, [nativeBootstrap]);

  async function action(load, type) {
    if (type === 'deliver' && !window.confirm('¿Confirmas que esta carga fue entregada?')) return;
    setBusyId(load.id); setNotice('');
    try {
      await postAction(`/api/mobile/loads/${load.id}/${type}`);
      await refresh(); setNotice('Estado de la carga actualizado.');
    } catch (error) { setNotice(error.message); }
    finally { setBusyId(null); }
  }
  async function loadMore() {
    if (!hasMore || loadingMore) return;
    setLoadingMore(true);
    try {
      const nextPage = page + 1;
      const data = await request('/api/mobile/loads?page=' + nextPage);
      setLoads(current => [...current, ...(data.loads || [])]);
      setPage(nextPage); setHasMore(Boolean(data.has_more));
    } catch (error) { setNotice(error.message); }
    finally { setLoadingMore(false); }
  }
  async function logout() { window.location.assign('/logout'); }

  const visible = useMemo(() => loads.filter(load => `${load.reference} ${load.origin.city} ${load.destination.city} ${load.status}`.toLowerCase().includes(query.toLowerCase())), [loads,query]);
  const active = loads.filter(load => ['asignada','en_transito'].includes(load.status)).length;
  const delivered = loads.filter(load => load.status === 'entregada').length;

  if (nativeBootstrap && !nativeReady) return <main className="login-wrap"><div className="login-card"><p className="eyebrow">BUFALO MÓVIL</p><h1>Conectando…</h1><p className="muted">Abriendo tu servidor de operaciones.</p></div></main>;
  if (nativeBootstrap) return <NativeServerSetup />;
  if (!user) return <><Login onLogin={refresh} busy={busy} setBusy={setBusy} setNotice={setNotice}/>{notice && <div className="toast error">{notice}</div>}</>;
  const roleName = {admin:'Administrador',publicador:'Publicador',chofer:'Chofer'}[user.role] || 'Usuario';
  const title = {home:'Inicio',loads:'Cargas',invoices:'Facturas',addresses:'Direcciones',profile:'Mi perfil',admin:'Administración'}[view] || 'Inicio';
  return <div className="app-shell"><header className="app-header"><a className="brand" href="/mobile/" onClick={e=>{e.preventDefault();setView('home')}}><span className="brand-mark small-mark">B</span><span>BUFALO<small>OPERACIONES</small></span></a><button className="avatar" onClick={()=>setView('profile')} aria-label="Abrir perfil">{user.profile_photo?<img src={user.profile_photo} alt=""/>:user.name.slice(0,1).toUpperCase()}</button></header>
    <main className="content"><div className="greeting"><div><p className="eyebrow">{roleName}</p><h1>{view==='home'?`Hola, ${user.name.split(' ')[0]}`:title}</h1><p className="muted">{view==='home'?'Aquí tienes el estado de tu operación.':'Consulta y gestiona tu actividad.'}</p></div>{installPrompt && <button className="install-button" onClick={async()=>{await installPrompt.prompt();setInstallPrompt(null)}}>Instalar app</button>}</div>
      {notice && <div className="notice" role="status"><span>{notice}</span><button onClick={()=>setNotice('')} aria-label="Cerrar">×</button></div>}
      {view==='home' && <><section className="metrics"><button onClick={()=>setView('loads')}><strong>{loads.length}</strong><span>Cargas visibles</span></button><button onClick={()=>setView('loads')}><strong>{active}</strong><span>En operación</span></button><button onClick={()=>setView('loads')}><strong>{delivered}</strong><span>Entregadas</span></button></section>
        <section className="section-heading"><div><p className="eyebrow">ACTIVIDAD</p><h2>{user.role==='chofer'?'Cargas para ti':'Últimas cargas'}</h2></div><button className="text-button" onClick={()=>setView('loads')}>Ver todas <span>→</span></button></section>
        {user.role==='publicador' && <div className="quick-actions"><a href={links.createLoad}><span>＋</span><b>Publicar carga</b></a><a href="/red-choferes"><span>♧</span><b>Mi red de choferes</b></a><a href="/choferes?orden=puntaje"><span>★</span><b>Buscar choferes mejor valorados</b></a><a href={links.invoices}><span>＄</span><b>Facturas</b></a><a href={links.addresses}><span>⌖</span><b>Direcciones</b></a></div>}
        {user.role==='admin' && <div className="quick-actions"><a href={links.admin}><span>▦</span><b>Panel de administración</b></a><a href={links.invoices}><span>＄</span><b>Facturas</b></a><a href={links.addresses}><span>⌖</span><b>Direcciones</b></a></div>}
        {user.role==='chofer' && <div className="quick-actions"><a href={links.addresses}><span>⌖</span><b>Direcciones</b></a><a href={links.invoices}><span>＄</span><b>Facturas</b></a></div>}
        {loads.slice(0,3).map(load=><LoadCard key={load.id} load={load} role={user.role} onAction={action} busyId={busyId}/>)}
      </>}
      {view==='loads' && <><div className="search"><span>⌕</span><input aria-label="Buscar cargas" placeholder="Buscar por referencia o ciudad" value={query} onChange={e=>setQuery(e.target.value)}/><button onClick={()=>refresh().catch(err=>setNotice(err.message))} aria-label="Actualizar">↻</button></div>
        {visible.length ? visible.map(load=><LoadCard key={load.id} load={load} role={user.role} onAction={action} busyId={busyId}/>) : <div className="empty"><span>⌕</span><h3>No hay cargas para mostrar</h3><p>Cuando haya actividad disponible, aparecerá aquí.</p>{user.role==='publicador' && <a className="button primary" href={links.createLoad}>Publicar carga</a>}</div>}
        {hasMore && <button className="button more-button full" disabled={loadingMore} onClick={loadMore}>{loadingMore?'Cargando…':'Cargar más'}</button>}
        {user.role==='publicador' && <a className="button primary full create-cta" href={links.createLoad}>＋ Publicar una carga</a>}
      </>}
      {['invoices','addresses','profile','admin'].includes(view) && <section className="feature-card">{view==='profile'&&user.profile_photo?<img className="profile-photo" src={user.profile_photo} alt={`Foto de ${user.name}`}/>:<div className="feature-icon">{view==='invoices'?'＄':view==='addresses'?'⌖':view==='profile'?'◉':'▦'}</div>}<p className="eyebrow">{roleName}</p><h2>{view==='profile'?user.name:title}</h2><p className="muted">{view==='profile'?user.email:'Este módulo utiliza tu sesión y los permisos de tu cuenta.'}</p>{view==='profile'&&user.role==='chofer'&&<p className="driver-rating">{user.rating_count?`${Number(user.driver_rating).toFixed(1)} ★ · ${user.rating_count} viajes calificados`:'Aún no tienes calificaciones'}</p>}{view==='profile'?<><a className="button primary full" href={links.profile}>Editar perfil</a>{Capacitor.isNativePlatform() && <button className="button full change-server" onClick={()=>window.location.assign('https://localhost/?changeServer=1')}>Cambiar servidor</button>}<button className="button full logout" onClick={logout}>Cerrar sesión</button></>:<a className="button primary full" href={links[view]}>Abrir {title.toLowerCase()}</a>}</section>}
    </main>
    <nav className="tabbar" aria-label="Navegación móvil"><button className={view==='home'?'selected':''} onClick={()=>setView('home')}><span>⌂</span>Inicio</button><button className={view==='loads'?'selected':''} onClick={()=>setView('loads')}><span>▤</span>Cargas</button><button className={view==='invoices'?'selected':''} onClick={()=>setView('invoices')}><span>＄</span>Facturas</button><button className={view==='profile'?'selected':''} onClick={()=>setView('profile')}><span>◉</span>Cuenta</button></nav>
  </div>;
}

createRoot(document.getElementById('root')).render(<React.StrictMode><App/></React.StrictMode>);

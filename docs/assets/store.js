'use strict';
(() => {
const API='https://dcs-store.corekeeper-research.workers.dev';
const $=id=>document.getElementById(id);
let config,signup=true,user=null,authWidget,checkoutWidget,authToken='',checkoutToken='',busy=false;
let token=sessionStorage.getItem('dcs-session')||'';
const requested=new URLSearchParams(location.search).get('sku')||'automation';
const names={pending:'Pendiente de completar',processing:'Procesando',paid:'Pago de prueba confirmado',trialing:'Prueba iniciada',active:'Suscripción de prueba activa',canceled:'Suscripción cancelada',failed:'No completado',past_due:'Pago de prueba pendiente',expired:'Vencido',paused:'En pausa'};
function notice(message,error=false){$('notice').textContent=message;$('notice').hidden=!message;$('notice').classList.toggle('store-error',error);}
async function api(path,body){
 const headers={};if(token)headers.Authorization=`Bearer ${token}`;if(body)headers['Content-Type']='application/json';
 const res=await fetch(API+path,{method:body?'POST':'GET',headers,body:body?JSON.stringify(body):undefined,cache:'no-store',signal:AbortSignal.timeout(20000)});
 const data=await res.json();if(!res.ok){if(res.status===401&&path!=='/auth/login'){token='';sessionStorage.removeItem('dcs-session');user=null;showAuth();}throw new Error(data.error||'No se pudo completar la solicitud.');}return data;
}
function safeError(e){notice(e.name==='TimeoutError'?'La conexión tardó demasiado. Intenta de nuevo.':e.message||'Error de conexión.',true);}
function buttons(){ $('auth-submit').disabled=busy||!authToken;$('checkout-submit').disabled=busy||!checkoutToken; }
function renderSecurity(which){
 if(!config||!window.turnstile)return;
 const auth=which==='auth',old=auth?authWidget:checkoutWidget;
 if(old!==undefined)window.turnstile.remove(old);
 if(auth)authToken='';else checkoutToken='';buttons();
 const id=window.turnstile.render(`#${which}-security`,{sitekey:config.sitekey,action:auth?(signup?'signup':'login'):'checkout',theme:'dark',language:'es',callback:value=>{if(auth)authToken=value;else checkoutToken=value;buttons();},'expired-callback':()=>{if(auth)authToken='';else checkoutToken='';buttons();},'error-callback':()=>{if(auth)authToken='';else checkoutToken='';buttons();notice('No se pudo cargar la verificación. Revisa tu conexión y vuelve a intentarlo.',true);}});
 if(auth)authWidget=id;else checkoutWidget=id;
}
function resetSecurity(which){const id=which==='auth'?authWidget:checkoutWidget;if(which==='auth')authToken='';else checkoutToken='';if(window.turnstile&&id!==undefined)window.turnstile.reset(id);buttons();}
function setMode(value){signup=value;$('signup-tab').setAttribute('aria-pressed',String(value));$('login-tab').setAttribute('aria-pressed',String(!value));$('name-field').hidden=!value;$('name').required=value;$('terms-field').hidden=!value;$('terms').required=value;$('auth-submit').textContent=value?'Crear cuenta':'Entrar';$('password').autocomplete=value?'new-password':'current-password';renderSecurity('auth');notice('');}
function showAuth(){ $('auth-panel').hidden=false;$('checkout-panel').hidden=true;$('orders-panel').hidden=true;renderSecurity('auth'); }
function showUser(){ $('auth-panel').hidden=true;$('checkout-panel').hidden=false;$('orders-panel').hidden=false;$('greeting').textContent=`Hola, ${user.name}. Tu correo de cuenta es ${user.email}.`; $('password').value='';renderSecurity('checkout');return loadOrders(); }
function summary(){const p=config.products.find(p=>p.sku===$('product').value);if(!p)return;$('summary-name').textContent=p.name;$('summary-price').textContent=p.trialDays?'$0 USD hoy':`${new Intl.NumberFormat('es-MX',{style:'currency',currency:'USD',maximumFractionDigits:0}).format(p.amount/100)} USD`;$('summary-frequency').textContent=p.trialDays?'1 día gratis; después, $12 USD/mes de prueba.':'Pago único de prueba.';$('summary-detail').textContent=p.trialDays?'Este plan reproduce una suscripción con un día gratuito. La renovación también es simulada; no se cobra dinero real ni se activa un servicio.':'El precio mostrado reproduce el servicio del catálogo. Este recorrido no genera un encargo ni un cargo real.';}
async function loadOrders(){const {orders}=await api('/orders');$('orders').replaceChildren();if(!orders.length){$('orders').textContent='Aún no tienes pedidos.';return;}for(const o of orders){const row=document.createElement('div');row.className='store-order';const title=document.createElement('strong');title.textContent=config.products.find(p=>p.sku===o.sku)?.name||o.sku;const status=document.createElement('span');status.className='store-status';status.textContent=names[o.status]||o.status;const date=document.createElement('small');date.textContent=new Date(o.created_at).toLocaleString('es-MX');const ref=document.createElement('small');ref.textContent=`Pedido ${o.id}`;row.append(title,status,date,ref);$('orders').append(row);}}
$('signup-tab').onclick=()=>setMode(true);$('login-tab').onclick=()=>setMode(false);
$('product').onchange=()=>{summary();const u=new URL(location.href);u.searchParams.set('sku',$('product').value);history.replaceState(null,'',u);};
$('auth-form').onsubmit=async e=>{e.preventDefault();if(busy||!authToken)return;busy=true;buttons();notice('');try{const data=await api(signup?'/auth/signup':'/auth/login',{name:$('name').value,email:$('email').value,password:$('password').value,terms:$('terms').checked,turnstile:authToken});token=data.token;sessionStorage.setItem('dcs-session',token);user=data.user;await showUser();notice(signup?'Cuenta creada. Ya puedes elegir tu servicio.':'Sesión iniciada.');}catch(e){safeError(e);resetSecurity('auth');}finally{busy=false;buttons();}};
$('checkout-form').onsubmit=async e=>{e.preventDefault();if(busy||!checkoutToken)return;busy=true;buttons();notice('Preparando tu checkout de prueba…');try{const data=await api('/checkout',{sku:$('product').value,testConsent:$('test-consent').checked,turnstile:checkoutToken});const url=new URL(data.url);if(url.origin!=='https://buy.stripe.com'||!url.pathname.startsWith('/test_'))throw new Error('El enlace de checkout no es válido.');location.assign(url.href);}catch(e){safeError(e);busy=false;resetSecurity('checkout');buttons();}};
$('logout').onclick=async()=>{try{await api('/auth/logout',{});}catch{}token='';sessionStorage.removeItem('dcs-session');user=null;showAuth();notice('Sesión cerrada.');};
$('refresh').onclick=async()=>{try{await loadOrders();notice('Estados actualizados.');}catch(e){safeError(e);}};
async function init(){try{config=await api('/config');if(config.testMode!==true)throw new Error('Configuración de pruebas no disponible.');for(const p of config.products){const option=document.createElement('option');option.value=p.sku;option.textContent=p.trialDays?`${p.name} · $0 hoy`:`${p.name} · $${p.amount/100} USD`;$('product').append(option);}$('product').value=config.products.some(p=>p.sku===requested)?requested:'automation';summary();if(token){try{user=(await api('/me')).user;await showUser();}catch{showAuth();}}else showAuth();if(new URLSearchParams(location.search).get('checkout')==='returned')notice('Volviste de Stripe. Consulta el estado confirmado en tus pedidos; puede tardar unos segundos.');}catch(e){safeError(e);}}
// Both external script loading and API discovery must finish before rendering.
window.addEventListener('load',()=>{if(config)renderSecurity(user?'checkout':'auth');});
init();
})();

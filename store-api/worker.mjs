// Test-only storefront. No Stripe API credential and no card-data endpoint.
export const CATALOG = Object.freeze({
  automation: {name:'Automation Management',amount:1400,link:'https://buy.stripe.com/test_cNi00kf2X21D5848kH3cc00',paymentLink:'plink_1UZkBzJ6LGLQ4OOcecfpAeF2'},
  database: {name:'Customer Database Analysis',amount:1000,link:'https://buy.stripe.com/test_dRm9AUcUPcGh6c88kH3cc01',paymentLink:'plink_1UZkC9J6LGLQ4OOcPGvAbbjO'},
  debugging: {name:'Script & Code Debugging',amount:800,link:'https://buy.stripe.com/test_aFa28saMH5dPbwseJ53cc02',paymentLink:'plink_1UZkCAJ6LGLQ4OOcd4EfLi0z'},
  data: {name:'Customer Data Analysis',amount:600,link:'https://buy.stripe.com/test_7sY7sMg710Xz2ZW44r3cc03',paymentLink:'plink_1UZkCBJ6LGLQ4OOcba1moWW8'},
  consulting: {name:'Technical Consulting (1:1)',amount:1200,link:'https://buy.stripe.com/test_9B67sM6wr9u57gc30n3cc04',paymentLink:'plink_1UZkCCJ6LGLQ4OOcVR4Le0lu'},
  trial: {name:'Prueba de 1 día',amount:1200,link:'https://buy.stripe.com/test_6oUeVebQL9u52ZW9oL3cc05',paymentLink:'plink_1UZkCDJ6LGLQ4OOcL3D19cqc'}
});
const ORIGIN='https://diluidocognit.store';
const enc=new TextEncoder();
const hex=b=>Array.from(new Uint8Array(b),n=>n.toString(16).padStart(2,'0')).join('');
const sha=async s=>hex(await crypto.subtle.digest('SHA-256',enc.encode(s)));
const random=()=>hex(crypto.getRandomValues(new Uint8Array(32)));
const fail=(status,message)=>{throw Object.assign(new Error(message),{status});};
export async function passwordHash(password,salt){
  const key=await crypto.subtle.importKey('raw',enc.encode(password),'PBKDF2',false,['deriveBits']);
  return hex(await crypto.subtle.deriveBits({name:'PBKDF2',salt:enc.encode(salt),iterations:100000,hash:'SHA-256'},key,256));
}
async function same(a,b){
  // Verify HMAC signatures through Web Crypto rather than string comparison.
  const key=await crypto.subtle.importKey('raw',enc.encode('constant-time-comparison'),{name:'HMAC',hash:'SHA-256'},false,['sign','verify']);
  const sig=await crypto.subtle.sign('HMAC',key,enc.encode(a));
  return crypto.subtle.verify('HMAC',key,sig,enc.encode(b));
}
export async function verifyStripe(body,header,secret,now=Date.now()){
  if(!secret||!header)return false;
  const parts=header.split(',').map(s=>s.trim().split('='));
  const time=parts.find(p=>p[0]==='t')?.[1];
  if(!/^\d+$/.test(time||'')||Math.abs(now/1000-Number(time))>300)return false;
  const key=await crypto.subtle.importKey('raw',enc.encode(secret),{name:'HMAC',hash:'SHA-256'},false,['verify']);
  for(const [,value] of parts.filter(p=>p[0]==='v1')){
    if(!/^[a-f0-9]{64}$/i.test(value||''))continue;
    const sig=Uint8Array.from(value.match(/../g),s=>parseInt(s,16));
    if(await crypto.subtle.verify('HMAC',key,sig,enc.encode(`${time}.${body}`)))return true;
  }
  return false;
}
async function readBody(req,max=8192){
  const reader=req.body?.getReader();if(!reader)return '';
  let size=0;const chunks=[];
  for(;;){const {done,value}=await reader.read();if(done)break;size+=value.byteLength;if(size>max){await reader.cancel();fail(413,'Solicitud demasiado grande.');}chunks.push(value);}
  const buffer=new Uint8Array(size);let offset=0;for(const c of chunks){buffer.set(c,offset);offset+=c.length;}
  return new TextDecoder().decode(buffer);
}
async function jsonBody(req){
  if(!req.headers.get('content-type')?.startsWith('application/json'))fail(415,'Se requiere JSON.');
  try{const data=JSON.parse(await readBody(req));if(!data||typeof data!=='object'||Array.isArray(data))fail(400,'Se requiere un objeto JSON.');return data;}catch(e){if(e.status)throw e;fail(400,'JSON inválido.');}
}
async function rate(env,key,limit,now){
  const bucket=Math.floor(now/600000);const id=`${key}:${bucket}`;
  const r=await env.DB.prepare('INSERT INTO rate_limits(key,count,expires_at) VALUES(?,1,?) ON CONFLICT(key) DO UPDATE SET count=count+1 RETURNING count').bind(id,now+600000).first();
  if(r.count>limit)fail(429,'Demasiados intentos. Espera unos minutos.');
}
async function challenge(req,env,token,action){
  if(typeof token!=='string'||!token||token.length>2048)fail(403,'Completa la verificación de seguridad.');
  const res=await fetch('https://challenges.cloudflare.com/turnstile/v0/siteverify',{method:'POST',body:new URLSearchParams({secret:env.TURNSTILE_SECRET,response:token,remoteip:req.headers.get('CF-Connecting-IP')||''}),signal:AbortSignal.timeout(10000)});
  if(!res.ok)fail(503,'La verificación no está disponible. Intenta de nuevo.');
  const d=await res.json();if(!d.success||d.hostname!=='diluidocognit.store'||d.action!==action)fail(403,'Verificación vencida o inválida. Intenta de nuevo.');
}
async function userFor(req,env,now){
  const token=req.headers.get('authorization')?.match(/^Bearer ([a-f0-9]{64})$/)?.[1];if(!token)fail(401,'Inicia sesión para continuar.');
  const row=await env.DB.prepare('SELECT u.id,u.name,u.email FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=? AND s.expires_at>?').bind(await sha(token),now).first();
  if(!row)fail(401,'Tu sesión venció. Inicia sesión de nuevo.');return row;
}
async function newSession(env,user,now){
  const token=random();await env.DB.prepare('INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)').bind(await sha(token),user.id,now+7200000).run();
  return {user:{id:user.id,name:user.name,email:user.email},token};
}
async function webhook(req,env,now){
  const raw=await readBody(req,262144);
  if(!await verifyStripe(raw,req.headers.get('stripe-signature'),env.STRIPE_WEBHOOK_SECRET,now))fail(400,'Firma inválida.');
  let event;try{event=JSON.parse(raw);}catch{fail(400,'Evento inválido.');}
  if(event.livemode!==false)fail(400,'Solo eventos de prueba.');
  if(typeof event.id!=='string'||!event.data?.object)fail(400,'Evento inválido.');
  if(await env.DB.prepare('SELECT id FROM events WHERE id=?').bind(event.id).first())return {received:true};
  const o=event.data.object;let statements=[];
  if(['checkout.session.completed','checkout.session.async_payment_succeeded','checkout.session.async_payment_failed'].includes(event.type)){
    const order=await env.DB.prepare('SELECT * FROM orders WHERE id=?').bind(o.client_reference_id||'').first();
    if(order){
      const product=CATALOG[order.sku];
      if(o.livemode!==false||o.payment_link!==product.paymentLink||o.currency!=='usd')fail(400,'El evento no corresponde al pedido.');
      if(order.stripe_session&&order.stripe_session!==o.id)fail(400,'Pedido ya vinculado.');
      const isTrial=order.sku==='trial';
      if(o.mode!==(isTrial?'subscription':'payment')||o.amount_total!==(isTrial?0:product.amount))fail(400,'Importe o modalidad inválidos.');
      let status='processing';
      if(event.type==='checkout.session.async_payment_failed')status='failed';
      else if(isTrial&&o.payment_status==='no_payment_required'&&o.subscription)status='trialing';
      else if(!isTrial&&o.payment_status==='paid')status='paid';
      // An old failure must never downgrade a paid order.
      statements.push(env.DB.prepare("UPDATE orders SET status=CASE WHEN status IN ('paid','trialing','active','canceled','expired') THEN status ELSE ? END,stripe_session=?,subscription_id=?,updated_at=? WHERE id=?").bind(status,o.id,typeof o.subscription==='string'?o.subscription:null,now,order.id));
    }
  } else if(['customer.subscription.updated','customer.subscription.deleted'].includes(event.type)){
    const known={trialing:'trialing',active:'active',canceled:'canceled',unpaid:'failed',past_due:'past_due',incomplete_expired:'expired',paused:'paused'};
    const status=event.type.endsWith('.deleted')?'canceled':known[o.status];
    if(status)statements.push(env.DB.prepare('UPDATE orders SET status=?,updated_at=? WHERE subscription_id=? AND updated_at<=?').bind(status,event.created*1000,o.id,event.created*1000));
  }
  // Update and deduplicate atomically. A retry after a transient failure remains processable.
  statements.push(env.DB.prepare('INSERT OR IGNORE INTO events(id,created_at) VALUES(?,?)').bind(event.id,now));
  await env.DB.batch(statements);return {received:true};
}
export async function handle(req,env,now=Date.now()){
  const url=new URL(req.url);const path=url.pathname;
  if(path==='/stripe/webhook'&&req.method==='POST')return webhook(req,env,now);
  if(req.headers.get('origin')!==ORIGIN)fail(403,'Origen no permitido.');
  if(path==='/config'&&req.method==='GET')return {testMode:true,sitekey:env.TURNSTILE_SITEKEY,products:Object.entries(CATALOG).map(([sku,p])=>({sku,name:p.name,amount:p.amount,currency:'usd',trialDays:sku==='trial'?1:0}))};
  if(path==='/auth/signup'||path==='/auth/login'){
    if(req.method!=='POST')fail(405,'Método no permitido.');
    await rate(env,`auth:${req.headers.get('CF-Connecting-IP')||'unknown'}`,12,now);
    const d=await jsonBody(req);const email=String(d.email||'').trim().toLowerCase();const password=d.password;
    if(!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)||email.length>254||typeof password!=='string'||password.length<12||password.length>128)fail(400,'Usa un correo válido y una contraseña de 12 a 128 caracteres.');
    const signup=path.endsWith('signup');await challenge(req,env,d.turnstile,signup?'signup':'login');
    if(signup){
      const name=String(d.name||'').trim();if(name.length<2||name.length>100||d.terms!==true)fail(400,'Escribe tu nombre y acepta las condiciones.');
      if(await env.DB.prepare('SELECT id FROM users WHERE email=?').bind(email).first())fail(409,'No se pudo crear la cuenta. Si ya tienes una, inicia sesión.');
      const salt=random(),id=crypto.randomUUID();
      await env.DB.prepare('INSERT INTO users(id,email,name,password_hash,salt,created_at) VALUES(?,?,?,?,?,?)').bind(id,email,name,await passwordHash(password,salt),salt,now).run();
      return newSession(env,{id,email,name},now);
    }
    const user=await env.DB.prepare('SELECT * FROM users WHERE email=?').bind(email).first();
    const hash=await passwordHash(password,user?.salt||'nonexistent-account-salt');
    if(!await same(hash,user?.password_hash||'invalid-hash'))fail(401,'Correo o contraseña incorrectos.');
    return newSession(env,user,now);
  }
  const user=await userFor(req,env,now);
  if(path==='/me'&&req.method==='GET')return {user};
  if(path==='/auth/logout'&&req.method==='POST'){
    await env.DB.prepare('DELETE FROM sessions WHERE token_hash=?').bind(await sha(req.headers.get('authorization').slice(7))).run();return {ok:true};
  }
  if(path==='/orders'&&req.method==='GET'){
    const {results}=await env.DB.prepare('SELECT id,sku,status,created_at,updated_at FROM orders WHERE user_id=? ORDER BY created_at DESC LIMIT 50').bind(user.id).all();return {orders:results};
  }
  if(path==='/checkout'&&req.method==='POST'){
    await rate(env,`checkout:${user.id}`,10,now);
    const d=await jsonBody(req);if(!Object.hasOwn(CATALOG,d.sku)||d.testConsent!==true)fail(400,'Elige un servicio y confirma el modo de pruebas.');
    await challenge(req,env,d.turnstile,'checkout');
    const p=CATALOG[d.sku];
    // Resume the same pending order; do not create duplicates after a retry.
    const existing=await env.DB.prepare("SELECT id FROM orders WHERE user_id=? AND sku=? AND status='pending' AND created_at>? ORDER BY created_at DESC LIMIT 1").bind(user.id,d.sku,now-1800000).first();
    const id=existing?.id||crypto.randomUUID();
    if(!existing)await env.DB.prepare('INSERT INTO orders(id,user_id,sku,created_at,updated_at) VALUES(?,?,?,?,?)').bind(id,user.id,d.sku,now,now).run();
    const target=new URL(p.link);if(!target.pathname.startsWith('/test_'))fail(503,'Checkout no disponible.');
    target.searchParams.set('client_reference_id',id);target.searchParams.set('prefilled_email',user.email);
    return {url:target.href,order:id,testMode:true};
  }
  fail(404,'Ruta no encontrada.');
}
export default {async fetch(req,env,ctx){
  const id=crypto.randomUUID(),start=Date.now(),origin=req.headers.get('origin');
  const headers={'Content-Type':'application/json; charset=utf-8','Cache-Control':'no-store','X-Content-Type-Options':'nosniff','X-Request-Id':id,'Vary':'Origin'};
  if(origin===ORIGIN){headers['Access-Control-Allow-Origin']=ORIGIN;headers['Access-Control-Allow-Headers']='Authorization, Content-Type';headers['Access-Control-Allow-Methods']='GET, POST, OPTIONS';headers['Access-Control-Expose-Headers']='X-Request-Id';}
  let status=200,body;
  try{if(req.method==='OPTIONS'){if(origin!==ORIGIN)fail(403,'Origen no permitido.');return new Response(null,{status:204,headers});}body=await handle(req,env);}
  catch(e){status=e.status||500;body={error:e.status?e.message:'No se pudo completar la solicitud. Intenta de nuevo.',requestId:id};}
  console.log(JSON.stringify({requestId:id,path:new URL(req.url).pathname,method:req.method,status,durationMs:Date.now()-start}));
  // No names, email addresses, request bodies, credentials or payment data in logs.
  if(ctx&&Math.random()<0.01)ctx.waitUntil(env.DB.batch([env.DB.prepare('DELETE FROM sessions WHERE expires_at<?').bind(start),env.DB.prepare('DELETE FROM rate_limits WHERE expires_at<?').bind(start)]).catch(()=>{}));
  return new Response(JSON.stringify(body),{status,headers});
}};

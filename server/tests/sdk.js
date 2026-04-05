var SentinelSDK=function(t){
    "use strict";function n(t,e){const r=c();return(n=function(t,n){return r[t-=0]})(t,e)}const e=n,r=function(){let t=!0;return function(e,r){const o=t?function(){if(r){const t=r[n(3)](e,arguments);return r=null,t}}:function(){};return t=!1,o}}(),o=r(void 0,(function(){const t=n;return o[t(0)]().search("(((.+)+)+)+$").toString()[t(4)](o).search("(((.+)+)+)+$")}));o();const i=[];function c(){const t=["toString","Stringified UUID is invalid","push","apply","constructor","toLowerCase"];return(c=function(){return t})()}for(let t=0;t<256;++t)i[e(2)]((t+256)[e(0)](16).slice(1));const s=function(){let t=!0;return function(n,e){const r=t?function(){if(e){const t=e.apply(n,arguments);return e=null,t}}:function(){};return t=!1,r}}(),u=s(void 0,(function(){const t=l;return u[t(3)]()[t(1)](t(5)).toString()[t(4)](u)[t(1)]("(((.+)+)+)+$")}));let a;function f(){const t=["getRandomValues","search","crypto.getRandomValues() not supported. See https://github.com/uuidjs/uuid#getrandomvalues-not-supported","toString","constructor","(((.+)+)+)+$"];return(f=function(){return t})()}function l(t,n){const e=f();return(l=function(t,n){return e[t-=0]})(t,n)}u();const d=new Uint8Array(16);const h=g,p=function(){let t=!0;return function(n,e){const r=t?function(){if(e){const t=e[g(6)](n,arguments);return e=null,t}}:function(){};return t=!1,r}}(),m=p(void 0,(function(){const t=g;return m[t(2)]()[t(3)]("(((.+)+)+)+$").toString()[t(5)](m)[t(3)](t(1))}));function g(t,n){const e=w();return(g=function(t,n){return e[t-=0]})(t,n)}function w(){const t=["randomUUID","(((.+)+)+)+$","toString","search","undefined","constructor","apply"];return(w=function(){return t})()}m();var y={randomUUID:typeof crypto!==h(4)&&crypto[h(0)]&&crypto[h(0)].bind(crypto)};const v=function(){let t=!0;return function(n,e){const r=t?function(){if(e){const t=e.apply(n,arguments);return e=null,t}}:function(){};return t=!1,r}}(),b=v(void 0,(function(){const t=S;return b[t(0)]()[t(2)](t(5)).toString()[t(1)](b)[t(2)]("(((.+)+)+)+$")}));function k(){const t=["toString","constructor","search"," is out of buffer bounds","randomUUID","(((.+)+)+)+$","Random bytes length must be >= 16","length"];return(k=function(){return t})()}function S(t,n){const e=k();return(S=function(t,n){return e[t-=0]})(t,n)}function C(t,n,r){const o=S;if(y[o(4)]&&!t)return y[o(4)]();const c=(t=t||{}).random??t.rng?.()??function(){const t=l;if(!a){if("undefined"==typeof crypto||!crypto.getRandomValues)throw new Error(t(2));a=crypto[t(0)].bind(crypto)}return a(d)}();if(c[o(7)]<16)throw new Error(o(6));return c[6]=15&c[6]|64,c[8]=63&c[8]|128,function(t,n=0){const r=e;return(i[t[n+0]]+i[t[n+1]]+i[t[n+2]]+i[t[n+3]]+"-"+i[t[n+4]]+i[t[n+5]]+"-"+i[t[n+6]]+i[t[n+7]]+"-"+i[t[n+8]]+i[t[n+9]]+"-"+i[t[n+10]]+i[t[n+11]]+i[t[n+12]]+i[t[n+13]]+i[t[n+14]]+i[t[n+15]])[r(5)]()}(c)}b();
const A=j;
function O(){const t=["cache","constructor","startEnforcement","length","userAgent","getEnforcementTokenSync","string","memory","fromCharCode","forceSync","data","getEnforcementToken","getRequirementsTokenBlocking","apply","documentElement","languages","encode","TextEncoder","_runCheck","getConfig","wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D","language","(((.+)+)+)+$","random","filter","errorPrefix","set","imul","has","hardwareConcurrency","join","data-build","timeOrigin","InstallTrigger","toString","_generateAnswerSync","stringify","map","search","requirementsSeed","location","solana","sid","proofofwork","required","gAAAAAB","_generateAnswerAsync","buildGenerateFailMessage","keys","initializeAndGatherData","src","getAttribute","then","_getAnswer","getRequirementsToken","maxAttempts","requestIdleCallback","scripts","createPRNG","jsHeapSizeLimit","dump","answers","_generateRequirementsTokenAnswerBlocking","match","now","get","substring"];return(O=function(){return t})()}
class _{
    ['answers']=new Map;
    ['maxAttempts']=5e5;
    ['requirementsSeed']=function(){
        const t=function(){
            let t=!0;return function(n,e){const r=t?function(){if(e){const t=e['apply'](n,arguments);return e=null,t}}:function(){};return t=!1,r}
        }(),n=t(this,(function(){
            return n['toString']()['search']("(((.+)+)+)+$").toString()['constructor'](n)['search']("(((.+)+)+)+$")}));return n(),""+Math.random()
    }();
    ['sid']=C();
    ['errorPrefix']='wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D';
    async['initializeAndGatherData'](t){
        this['_getAnswer'](t)
    }
    async['startEnforcement'](t){
        this['_getAnswer'](t)
    }
    ['getEnforcementTokenSync'](t){
        const n=A,e=this._getAnswer(t);return typeof e===n(6)?e:null
    }
    async['getEnforcementToken'](t,n){
        const e=A;return this._getAnswer(t,n?.[e(9)])
    }
    async['getRequirementsToken'](){
        console.info('seed:', this.requirementsSeed)
        return !this['answers']['has'](this.requirementsSeed)
        &&
        this['answers']['set'](this.requirementsSeed,this['_generateAnswerAsync'](this['requirementsSeed'],"0")),
        "gAAAAAC"+await this.answers['get'](this['requirementsSeed'])
    }
    ['getRequirementsTokenBlocking'](){return"gAAAAAC"+this['_generateRequirementsTokenAnswerBlocking']()}['_getAnswer'](t,n=!1){const e=A,r=e(45);if(!t?.[e(43)]?.[e(44)])return null;const{seed:o,difficulty:i}=t.proofofwork;if(typeof o!==e(6)||typeof i!==e(6))return null;const c=this.answers[e(65)](o);if(typeof c===e(6))return c;if(n){const t=this[e(35)](o,i),n=r+t;return this[e(61)][e(26)](o,n),n}return!this[e(61)][e(28)](o)&&this.answers.set(o,this[e(46)](o,i)),Promise.resolve()[e(52)]((async()=>{const t=e;return r+await this[t(61)][t(65)](o)})).then((t=>{const n=e;return this[n(61)][n(26)](o,t),t}))}
    ['_runCheck']=(t,n,e,r,attemptCount)=>{
        // console.info('runCheck:', t,n,e,r,attemptCount);
        r[3]=attemptCount,r[9]=Math.round(performance['now']()-t);
        const c=N(r),
        s=function(t){
            let e=2166136261;
            for(let r=0;r<t.length;r++)
                e^=t.charCodeAt(r),e=Math['imul'](e,16777619)>>>0;return e^=e>>>16,e=Math['imul'](e,2246822507)>>>0,e^=e>>>13,e=Math['imul'](e,3266489909)>>>0,e^=e>>>16,(e>>>0)['toString'](16).padStart(8,"0")
        }(n+c);
        return s['substring'](0,e['length'])<=e?c+"~S":null
    };
    ['buildGenerateFailMessage'](t){
        return this['errorPrefix']+N(String(t??"e"))
    }
    _generateAnswerSync(t,n){const e=A,r=performance[e(64)]();try{const o=this[e(19)]();for(let i=0;i<this[e(55)];i++){const c=this[e(18)](r,t,n,o,i);if(c)return c}}catch(t){return this[e(47)](t)}return this.buildGenerateFailMessage()}
    async['_generateAnswerAsync'](t,n){
        console.info('_generateAnswerAsync')
        const r=performance['now']();
        try{
            let o=null;const i=this['getConfig']();
            console.info('config:', i)
            for(let c=0;c<this.maxAttempts;c++){
                (!o||o.timeRemaining()<=0)
                &&
                (
                    o=await new Promise((resolve=>{
                        const cb=window['requestIdleCallback']||q;
                        cb((n=>{
                            resolve(n)
                        }),{timeout:10})
                    }))
                );
                const s=this['_runCheck'](r,t,n,i,c);
                if(s)
                    return s
            }
        }
        catch(t){
            return this['buildGenerateFailMessage'](t)
        }
        return this['buildGenerateFailMessage']()
    }
    ['_generateRequirementsTokenAnswerBlocking'](){const t=A;let n="e";const e=performance[t(64)]();try{const n=this[t(19)]();return n[3]=1,n[9]=performance[t(64)]()-e,N(n)}catch(t){n=N(String(t))}return this[t(25)]+n}
    ['getConfig'](){
        return[
            screen?.width+screen?.height,
            ""+new Date,
            performance?.['memory']?.['jsHeapSizeLimit'],
            Math?.['random'](),
            navigator['userAgent'],

            R(
                Array.from(document['scripts'])['map']((n=>n?.['src'])).filter((t=>t))
            ),
            (
                Array.from(document['scripts']||[])['map']((n=>n?.['src']?.['match']("c/[^/]*/_")))['filter']((n=>n?.['length']))[0]??[]
            )[0]??document['documentElement']['getAttribute']('data-build'),
            navigator['language'],
            navigator['languages']?.['join'](","),
            Math?.['random'](),

            T(),
            R(Object['keys'](document)),
            R(Object.keys(window)),
            performance['now'](),
            this.sid,

            [...new URLSearchParams(window['location']['search'])['keys']()].join(","),
            navigator?.['hardwareConcurrency'],
            performance['timeOrigin'],
            Number("ai"in window),
            Number('createPRNG'in window),

            Number('cache'in window),
            Number('data'in window),
            Number('solana'in window),
            Number('dump'in window),
            Number('InstallTrigger'in window)]
    }
}function R(t){const n=A;return t[Math.floor(Math[n(23)]()*t[n(3)])]}function T(){const t=R(Object.keys(Object.getPrototypeOf(navigator)));try{return t+"−"+navigator[t].toString()}catch{return""+t}}
    function N(t){
        // console.info('encode data:', t)
        const result = (t=JSON['stringify'](t),window['TextEncoder']?btoa(String['fromCharCode'](...(new TextEncoder)['encode'](t))):btoa(unescape(encodeURIComponent(t))))
        // console.info('encode result:', result)
        return result
    }function j(t,n){
        const e=O();return(j=function(t,n){return e[t-=0]})(t,n)
    }
    function q(t){return setTimeout((()=>{t({timeRemaining:()=>1,didTimeout:!1})}),0),0}
    var P=new _;
    function x(t,n){
        const e=U();return(x=function(t,n){return e[t-=0]})(t,n)
    }
    const E=function(){let t=!0;return function(n,e){const r=t?function(){if(e){const t=e[x(5)](n,arguments);return e=null,t}}:function(){};return t=!1,r}}(),M=E(void 0,(function(){const t=x;return M.toString()[t(1)](t(4))[t(2)]()[t(3)](M)[t(1)](t(4))}));function U(){const t=["get","search","toString","constructor","(((.+)+)+)+$","apply"];return(U=function(){return t})()}
    M();const I=new WeakMap;function D(t,n){I.set(t,n)}
    function $(t){return I[x(0)](t)}
    const F=function(){let t=!0;return function(n,e){const r=t?function(){if(e){const t=e[xt(19)](n,arguments);return e=null,t}}:function(){};return t=!1,r}}(),L=F(void 0,(function(){const t=xt;return L[t(2)]().search(t(26))[t(2)]().constructor(L).search(t(26))}));L();const J=0,G=1,W=2,z=3,H=4,V=5,B=6,Z=24,K=7,Q=8,Y=9,X=10,tt=11,nt=12,et=13,rt=14,ot=15,it=16,ct=17,st=18,ut=19,at=23,ft=20,lt=21,dt=22,ht=25,pt=26,mt=27,gt=28,wt=29,yt=30,vt=33,bt=34,kt=35,St=new Map;let Ct=0,At=Promise.resolve();function Ot(t){const n=xt,e=At[n(13)](t,t);return At=e[n(13)]((()=>{}),(()=>{})),e}async function _t(){const t=xt;for(;St[t(25)](Y).length>0;){const[n,...e]=St[t(25)](Y).shift(),r=St.get(n)(...e);r&&typeof r[t(13)]===t(17)&&await r,Ct++}}function Rt(t,n){const e=xt;let r="";for(let o=0;o<t[e(7)];o++)r+=String[e(6)](t[e(22)](o)^n[e(22)](o%n[e(7)]));return r}function Tt(){const t=["abs","snapshot_dx","toString","src","splice","isArray","fromCharCode","length","clear","map","set","session_observer_vm_timeout","collector_dx","then","parse","match","filter","function","finally","apply","resolve","indexOf","charCodeAt","bind","catch","get","(((.+)+)+)+$"];return(Tt=function(){return t})()}function Nt(t){return Ot((()=>jt(t)))}function jt(t,n){return new Promise(((e,r)=>{const o=xt;void 0!==n&&(function(){const t=xt;St[t(8)](),St.set(J,Nt),St[t(10)](G,((n,e)=>St[t(10)](n,Rt(""+St[t(25)](n),""+St.get(e))))),St[t(10)](W,((n,e)=>St[t(10)](n,e))),St[t(10)](V,((n,e)=>{const r=t,o=St[r(25)](n);Array[r(5)](o)?o.push(St[r(25)](e)):St.set(n,o+St.get(e))})),St.set(mt,((n,e)=>{const r=t,o=St.get(n);Array.isArray(o)?o[r(4)](o[r(21)](St.get(e)),1):St[r(10)](n,o-St[r(25)](e))})),St.set(wt,((n,e,r)=>St.set(n,St[t(25)](e)<St[t(25)](r)))),St.set(vt,((n,e,r)=>{const o=t,i=Number(St[o(25)](e)),c=Number(St[o(25)](r));St[o(10)](n,i*c)})),St.set(kt,((n,e,r)=>{const o=t,i=Number(St.get(e)),c=Number(St[o(25)](r));St[o(10)](n,0===c?0:i/c)})),St[t(10)](B,((n,e,r)=>St.set(n,St[t(25)](e)[St.get(r)]))),St.set(K,((n,...e)=>St[t(25)](n)(...e[t(9)]((n=>St[t(25)](n)))))),St[t(10)](ct,((n,e,...r)=>{const o=t;try{const t=St[o(25)](e)(...r[o(9)]((t=>St[o(25)](t))));if(t&&typeof t.then===o(17))return t[o(13)]((t=>{St.set(n,t)})).catch((t=>{St.set(n,""+t)}));St[o(10)](n,t)}catch(t){St[o(10)](n,""+t)}})),St[t(10)](et,((n,e,...r)=>{const o=t;try{St[o(25)](e)(...r)}catch(t){St[o(10)](n,""+t)}})),St.set(Q,((n,e)=>St[t(10)](n,St[t(25)](e)))),St[t(10)](X,window),St[t(10)](tt,((n,e)=>St[t(10)](n,(Array.from(document.scripts||[])[t(9)]((n=>n?.[t(3)]?.[t(15)](St[t(25)](e))))[t(16)]((n=>n?.[t(7)]))[0]??[])[0]??null))),St[t(10)](nt,(n=>St[t(10)](n,St))),St[t(10)](rt,((n,e)=>St[t(10)](n,JSON[t(14)](""+St[t(25)](e))))),St.set(ot,((n,e)=>St[t(10)](n,JSON.stringify(St.get(e))))),St[t(10)](st,(n=>St[t(10)](n,atob(""+St[t(25)](n))))),St[t(10)](ut,(n=>St[t(10)](n,btoa(""+St[t(25)](n))))),St[t(10)](ft,((n,e,r,...o)=>St[t(25)](n)===St[t(25)](e)?St[t(25)](r)(...o):null)),St[t(10)](lt,((n,e,r,o,...i)=>Math[t(0)](St[t(25)](n)-St.get(e))>St[t(25)](r)?St[t(25)](o)(...i):null)),St[t(10)](at,((n,e,...r)=>void 0!==St[t(25)](n)?St[t(25)](e)(...r):null)),St[t(10)](Z,((n,e,r)=>St[t(10)](n,St.get(e)[St.get(r)][t(23)](St.get(e))))),St[t(10)](bt,((n,e)=>{const r=t;try{const t=St[r(25)](e);return Promise[r(20)](t)[r(13)]((t=>{St.set(n,t)}))}catch{return}})),St[t(10)](dt,((n,e)=>{const r=t,o=[...St[r(25)](Y)];return St[r(10)](Y,[...e]),_t()[r(24)]((t=>{St[r(10)](n,""+t)}))[r(18)]((()=>{St[r(10)](Y,o)}))})),St[t(10)](gt,(()=>{})),St.set(pt,(()=>{})),St[t(10)](ht,(()=>{}))}(),Ct=0,St.set(it,n));let i=!1;const c=setTimeout((()=>{!i&&(i=!0,r(new Error(xt(11))))}),6e4),s=t=>{i||(i=!0,clearTimeout(c),e(t))};St[o(10)](z,(t=>{s(btoa(""+t))})),St.set(H,(t=>{(t=>{i||(i=!0,clearTimeout(c),r(t))})(btoa(""+t))})),St[o(10)](yt,((t,n,e,r)=>{const i=o,c=Array[i(5)](r),s=c?e:[],u=(c?r:e)||[];St[i(10)](t,((...t)=>{const e=i,r=[...St[e(25)](Y)];if(c)for(let n=0;n<s[e(7)];n++){const r=s[n],o=t[n];St[e(10)](r,o)}return St[e(10)](Y,[...u]),_t()[e(13)]((()=>St.get(n)))[e(24)]((t=>""+t))[e(18)]((()=>{St.set(Y,r)}))}))}));try{St[o(10)](Y,JSON[o(14)](Rt(atob(t),""+St[o(25)](it)))),_t()[o(24)]((t=>{s(btoa(Ct+": "+t))}))}catch(t){s(btoa(Ct+": "+t))}}))}function qt(t){return t?.so??null}function Pt(t){return!0===t?.required}function xt(t,n){const e=Tt();return(xt=function(t,n){return e[t-=0]})(t,n)}function Et(t){const n=xt,e=qt(t);t&&Pt(e)&&e?.[n(12)]&&function(t,n){const e=$(t??{})??"";return Ot((()=>jt(n,e)))}(t,e.collector_dx)[n(24)]((()=>{}))}const Mt=Rn,Ut=function(){let t=!0;return function(n,e){const r=t?function(){if(e){const t=e[Rn(7)](n,arguments);return e=null,t}}:function(){};return t=!1,r}}(),It=Ut(void 0,(function(){const t=Rn;return It[t(3)]()[t(24)](t(0))[t(3)]()[t(13)](It)[t(24)](t(0))}));function Dt(){const t=["(((.+)+)+)+$","fromCharCode","resolve","toString","set","match","clear","apply","abs","function","get","map","push","constructor","parse","charCodeAt","log","isArray","stringify","from","scripts","then","shift","filter","search","indexOf","length","finally","bind","catch"];return(Dt=function(){return t})()}It();const $t=0,Ft=1,Lt=2,Jt=3,Gt=4,Wt=5,zt=6,Ht=24,Vt=7,Bt=8,Zt=9,Kt=10,Qt=11,Yt=12,Xt=13,tn=14,nn=15,en=16,rn=17,on=18,cn=19,sn=23,un=20,an=21,fn=22,ln=25,dn=26,hn=27,pn=28,mn=29,gn=30,wn=33,yn=34,vn=35,bn=new Map;let kn=0,Sn=Promise['resolve']();
    function Cn(t){
        const n=Sn['then'](t,t);return Sn=n.then((()=>{}),(()=>{})),n
    }
    async function An(){
        for(;bn['get'](Zt)['length']>0;){
            const[n,...e]=bn['get'](Zt)['shift'](),r=bn['get'](n)(...e);r&&typeof r['then']==='function'&&await r,kn++
        }
    }
    function On(t){
        return Cn((()=>new Promise(((n,e)=>{const r=Rn;let o=!1;setTimeout((()=>{o=!0,n(""+kn)}),500),bn[r(4)](Jt,(t=>{!o&&(o=!0,n(btoa(""+t)))})),bn.set(Gt,(t=>{!o&&(o=!0,e(btoa(""+t)))})),bn[r(4)](gn,((t,n,e,i)=>{const c=r,s=Array[c(17)](i),u=s?e:[],a=(s?i:e)||[];bn.set(t,((...t)=>{const e=c;if(o)return;const r=[...bn[e(10)](Zt)];if(s)for(let n=0;n<u[e(26)];n++){const e=u[n],r=t[n];bn.set(e,r)}
    return bn.set(Zt,[...a]),
    An()[e(21)]((()=>bn.get(n)))
    [e(29)]((t=>""+t))[e(27)]((()=>{bn.set(Zt,r)}))
}))}));
    try{
        bn[r(4)](Zt,JSON[r(14)](Tn(atob(t),""+bn[r(10)](en)))),
        An()[r(29)]((t=>{n(btoa(kn+": "+t))}))
    }catch(t){
        n(btoa(kn+": "+t))}}))))}
    function _n(t,n){
        console.info('_n:', t, n)
        return Cn(
          () =>
            new Promise((resolve, r) => {
              const
                i = $(t ?? {}) ?? "";
              ((function () {
                (bn['clear'](),
                  bn['set']($t, On),
                  bn['set'](Ft, (n, e) =>
                    bn.set(n, Tn("" + bn['get'](n), "" + bn['get'](e))),
                  ),
                  bn['set'](Lt, (n, e) => bn['set'](n, e)),
                  bn['set'](Wt, (n, e) => {
                    const o = bn['get'](n);
                    Array['isArray'](o)
                      ? o['push'](bn['get'](e))
                      : bn['set'](n, o + bn.get(e));
                  }),
                  bn['set'](hn, (n, e) => {
                    const
                      o = bn['get'](n);
                    Array['isArray'](o)
                      ? o.splice(o['indexOf'](bn['get'](e)), 1)
                      : bn['set'](n, o - bn['get'](e));
                  }),
                  bn['set'](mn, (n, e, r) =>
                    bn['set'](n, bn['get'](e) < bn.get(r)),
                  ),
                  bn.set(wn, (n, e, r) => {
                    const 
                      i = Number(bn['get'](e)),
                      c = Number(bn.get(r));
                    bn['set'](n, i * c);
                  }),
                  bn['set'](vn, (n, e, r) => {
                    const
                      i = Number(bn['get'](e)),
                      c = Number(bn['get'](r));
                    bn['set'](n, 0 === c ? 0 : i / c);
                  }),
                  bn['set'](zt, (n, e, r) =>
                    bn.set(n, bn['get'](e)[bn['get'](r)]),
                  ),
                  bn['set'](Vt, (n, ...e) =>{
                    const args = e['map']((n) => bn['get'](n))
                    const func = bn['get'](n)
                    console.info('call function:', func?.name, 'with args:', args)
                    const result = func(...args)
                    console.info('function result:',func, result)
                    return result
                  }),
                  bn.set(rn, (n, e, ...r) => {
                    try {
                      const t = bn['get'](e)(...r['map']((t) => bn['get'](t)));
                      if (t && typeof t.then === 'function')
                        return t['then']((t) => {
                          bn['set'](n, t);
                        })['catch']((t) => {
                          bn['set'](n, "" + t);
                        });
                      bn['set'](n, t);
                    } catch (t) {
                      bn['set'](n, "" + t);
                    }
                  }),
                  bn['set'](Xt, (n, e, ...r) => {
                    try {
                      bn['get'](e)(...r);
                    } catch (t) {
                      bn['set'](n, "" + t);
                    }
                  }),
                  bn['set'](Bt, (n, e) => bn['set'](n, bn['get'](e))),
                  bn.set(Kt, window),
                  bn.set(Qt, (n, e) =>
                    bn['set'](
                      n,
                      (Array['from'](document['scripts'] || [])
                        ['map']((n) => n?.src?.['match'](bn.get(e)))
                        ['filter']((n) => n?.['length'])[0] ?? [])[0] ?? null,
                    ),
                  ),
                  bn['set'](Yt, (n) => bn['set'](n, bn)),
                  bn['set'](tn, (n, e) =>
                    bn['set'](n, JSON['parse']("" + bn['get'](e))),
                  ),
                  bn.set(nn, (n, e) => bn['set'](n, JSON['stringify'](bn.get(e)))),
                  bn['set'](on, (n) => bn['set'](n, atob("" + bn['get'](n)))),
                  bn.set(cn, (n) => bn['set'](n, btoa("" + bn.get(n)))),
                  bn['set'](un, (n, e, r, ...o) =>
                    bn['get'](n) === bn.get(e) ? bn.get(r)(...o) : null,
                  ),
                  bn.set(an, (n, e, r, o, ...i) =>
                    Math['abs'](bn['get'](n) - bn['get'](e)) > bn['get'](r)
                      ? bn.get(o)(...i)
                      : null,
                  ),
                  bn['set'](sn, (n, e, ...r) =>
                    void 0 !== bn['get'](n) ? bn['get'](e)(...r) : null,
                  ),
                  bn['set'](Ht, (n, e, r) =>
                    bn['set'](n, bn['get'](e)[bn['get'](r)]['bind'](bn.get(e))),
                  ),
                  bn.set(yn, (n, e) => {
                    try {
                      const t = bn.get(e);
                      return Promise['resolve'](t).then((t) => {
                        bn['set'](n, t);
                      });
                    } catch (t) {
                      return;
                    }
                  }),
                  bn['set'](fn, (n, e) => {
                    const
                      o = [...bn['get'](Zt)];
                    return (
                      bn['set'](Zt, [...e]),
                      An()
                        ['catch']((t) => {
                          bn.set(n, "" + t);
                        })
                        ['finally'](() => {
                          bn.set(Zt, o);
                        })
                    );
                  }),
                  bn['set'](pn, () => {}),
                  bn.set(dn, () => {}),
                  bn.set(ln, () => {}));
              })(),
                (kn = 0),
                bn['set'](en, i));
              let c = !1;
              (setTimeout(() => {
                ((c = !0), resolve("" + kn));
              }, 500),
                bn['set'](Jt, (t) => {
                  !c && ((c = !0), resolve(btoa("" + t)));
                }),
                bn['set'](Gt, (t) => {
                  !c && ((c = !0), r(btoa("" + t)));
                }),
                bn['set'](gn, (t, n, e, r) => {
                  const 
                    s = Array['isArray'](r),
                    u = s ? e : [],
                    a = (s ? r : e) || [];
                  bn['set'](t, (...t) => {
                    if (c) return;
                    const r = [...bn['get'](Zt)];
                    if (s)
                      for (let n = 0; n < u.length; n++) {
                        const e = u[n],
                          r = t[n];
                        bn.set(e, r);
                      }
                    return (
                      bn['set'](Zt, [...a]),
                      An()
                        .then(() => bn['get'](n))

                        ['catch']((t) => "" + t)
                        .finally(() => {
                          bn.set(Zt, r);
                        })
                    );
                  });
                }));
              try {
                console.info('Tn:', atob(n), "" + bn['get'](en))
                const j = Tn(atob(n), "" + bn['get'](en));
                console.info('json:', j);
                (bn['set'](Zt, JSON.parse(j)),
                  An()['catch']((t) => {
                    console.info('error:', t)
                    resolve(btoa(kn + ": " + t));
                  }));
              } catch (t) {
                resolve(btoa(kn + ": " + t));
              }
            }),
        );
    }
        function Rn(t,n){const e=Dt();return(Rn=function(t,n){return e[t-=0]})(t,n)}
        function Tn(t,n){
            console.info('Tn input:', t, n);
            let r="";for(let o=0;o<t['length'];o++)r+=String['fromCharCode'](t['charCodeAt'](o)^n['charCodeAt'](o%n['length']));return r
        }
        var Nn="undefined"!=typeof globalThis?globalThis:"undefined"!=typeof window?window:"undefined"!=typeof global?global:"undefined"!=typeof self?self:{};var jn=Object.freeze({__proto__:null,commonjsGlobal:Nn,getAugmentedNamespace:function(t){if(t.__esModule)return t;var n=t.default;if("function"==typeof n){var e=function t(){if(this instanceof t){var e=[null];return e.push.apply(e,arguments),new(Function.bind.apply(n,e))}return n.apply(this,arguments)};e.prototype=n.prototype}else e={};return Object.defineProperty(e,"__esModule",{value:!0}),Object.keys(t).forEach((function(n){var r=Object.getOwnPropertyDescriptor(t,n);Object.defineProperty(e,n,r.get?r:{enumerable:!0,get:function(){return t[n]}})})),e},getDefaultExportFromCjs:function(t){return t&&t.__esModule&&Object.prototype.hasOwnProperty.call(t,"default")?t.default:t},getDefaultExportFromNamespaceIfNotNamed:function(t){return t&&Object.prototype.hasOwnProperty.call(t,"default")&&1===Object.keys(t).length?t.default:t},getDefaultExportFromNamespaceIfPresent:function(t){return t&&Object.prototype.hasOwnProperty.call(t,"default")?t.default:t}}),qn={},Pn={};function xn(){var t=["argument str must be a string","test","function","(((.+)+)+)+$","serialize","path","indexOf","slice","length","parse","domain","lax","split","substring","argument val is invalid","toString","option expires is invalid","string","; SameSite=Strict","encode","search","argument name is invalid","; Max-Age=","; Secure","decode","; SameSite=None","sameSite","trim","maxAge","; Expires=","; SameSite=Lax","apply","option domain is invalid","expires","; HttpOnly","option maxAge is invalid","toLowerCase","secure","option path is invalid"];return(xn=function(){return t})()}var En,Mn=Dn,Un=(En=!0,function(t,n){var e=En?function(){if(n){var e=n[Dn(31)](t,arguments);return n=null,e}}:function(){};return En=!1,e}),In=Un(void 0,(function(){var t=Dn;return In[t(15)]()[t(20)](t(3))[t(15)]().constructor(In).search(t(3))}));function Dn(t,n){var e=xn();return(Dn=function(t,n){return e[t-=0]})(t,n)}In(),Pn[Mn(9)]=function(t,n){var e=Mn;if(typeof t!==e(17))throw new TypeError(e(0));for(var r={},o=n||{},i=t[e(12)](";"),c=o[e(24)]||$n,s=0;s<i[e(8)];s++){var u=i[s],a=u[e(6)]("=");if(!(a<0)){var f=u[e(13)](0,a)[e(27)]();if(null==r[f]){var l=u[e(13)](a+1,u[e(8)])[e(27)]();'"'===l[0]&&(l=l[e(7)](1,-1)),r[f]=Jn(l,c)}}}return r},Pn[Mn(4)]=function(t,n,e){var r=Mn,o=e||{},i=o[r(19)]||Fn;if(typeof i!==r(2))throw new TypeError("option encode is invalid");if(!Ln.test(t))throw new TypeError(r(21));var c=i(n);if(c&&!Ln[r(1)](c))throw new TypeError(r(14));var s=t+"="+c;if(null!=o[r(28)]){var u=o.maxAge-0;if(isNaN(u)||!isFinite(u))throw new TypeError(r(35));s+=r(22)+Math.floor(u)}if(o[r(10)]){if(!Ln[r(1)](o[r(10)]))throw new TypeError(r(32));s+="; Domain="+o.domain}if(o[r(5)]){if(!Ln[r(1)](o[r(5)]))throw new TypeError(r(38));s+="; Path="+o[r(5)]}if(o.expires){if("function"!=typeof o[r(33)].toUTCString)throw new TypeError(r(16));s+=r(29)+o[r(33)].toUTCString()}if(o.httpOnly&&(s+=r(34)),o[r(37)]&&(s+=r(23)),o[r(26)]){switch(typeof o[r(26)]===r(17)?o[r(26)][r(36)]():o[r(26)]){case!0:s+=r(18);break;case r(11):s+=r(30);break;case"strict":s+=r(18);break;case"none":s+=r(25);break;default:throw new TypeError("option sameSite is invalid")}}return s};var $n=decodeURIComponent,Fn=encodeURIComponent,Ln=/^[\u0009\u0020-\u007e\u0080-\u00ff]+$/;function Jn(t,n){try{return n(t)}catch(n){return t}}var Gn=zn;function Wn(){var t=["undefined","indexOf","checkCookies","__assign","concat","call","search","join","replace","slice","(((.+)+)+)+$","constructor","cookie","test","toString","length","getHeader","headers","[WARN]: checkCookies was deprecated. It will be deleted in the new version. Use hasCookie instead.","removeCookies","hasOwnProperty","parse","commonjsGlobal","res","split","getCookies","false","deleteCookie","warn","prototype","getDefaultExportFromCjs","cookies","apply","stringify","null","serialize","setCookies","getCookie","setHeader","__rest","req","isArray","setCookie","true","Set-Cookie","[WARN]: setCookies was deprecated. It will be deleted in the new version. Use setCookie instead.","[WARN]: removeCookies was deprecated. It will be deleted in the new version. Use deleteCookie instead.","getOwnPropertySymbols","hasCookie"];return(Wn=function(){return t})()}function zn(t,n){var e=Wn();return(zn=function(t,n){return e[t-=0]})(t,n)}!function(t){var n,e=zn,r=(n=!0,function(t,e){var r=n?function(){if(e){var n=e[zn(32)](t,arguments);return e=null,n}}:function(){};return n=!1,r}),o=r(this,(function(){var t=zn;return o[t(14)]()[t(6)](t(10))[t(14)]()[t(11)](o)[t(6)](t(10))}));o();var i=jn[e(22)]&&jn[e(22)][e(3)]||function(){var t=e;return i=Object.assign||function(t){for(var n,e=zn,r=1,o=arguments.length;r<o;r++)for(var i in n=arguments[r])Object[e(29)][e(20)][e(5)](n,i)&&(t[i]=n[i]);return t},i[t(32)](this,arguments)},c=jn[e(22)]&&jn[e(22)][e(39)]||function(t,n){var r=e,o={};for(var i in t)Object[r(29)][r(20)][r(5)](t,i)&&n.indexOf(i)<0&&(o[i]=t[i]);if(null!=t&&"function"==typeof Object[r(47)]){var c=0;for(i=Object[r(47)](t);c<i[r(15)];c++)n[r(1)](i[c])<0&&Object[r(29)].propertyIsEnumerable[r(5)](t,i[c])&&(o[i[c]]=t[i[c]])}return o};Object.defineProperty(t,"__esModule",{value:!0}),t.checkCookies=t[e(48)]=t.removeCookies=t.deleteCookie=t[e(36)]=t.setCookie=t[e(37)]=t[e(25)]=void 0;var s=Pn,u=function(){return typeof window!==e(0)},a=function(t){var n=e;void 0===t&&(t="");try{var r=JSON[n(33)](t);return/^[\{\[]/[n(13)](r)?r:t}catch(n){return t}};t[e(25)]=function(t){var n,r=e;if(t&&(n=t[r(40)]),!u())return n&&n.cookies?n[r(31)]:n&&n.headers&&n[r(17)][r(12)]?(0,s.parse)(n[r(17)][r(12)]):{};for(var o={},i=document.cookie?document[r(12)].split("; "):[],c=0,a=i[r(15)];c<a;c++){var f=i[c][r(24)]("="),l=f[r(9)](1)[r(7)]("=");o[f[0]]=l}return o};t.getCookie=function(n,r){var o=(0,t[e(25)])(r)[n];if(void 0!==o)return function(t){var n=e;return t===n(43)||t!==n(26)&&("undefined"!==t?t===n(34)?null:t:void 0)}(function(t){return t?t[e(8)](/(%[0-9A-Z]{2})+/g,decodeURIComponent):t}(o))};t[e(42)]=function(t,n,r){var o,f,l,d=e;r&&(f=r[d(40)],l=r[d(23)],o=c(r,["req","res"]));var h=(0,s[d(35)])(t,a(n),i({path:"/"},o));if(u())document[d(12)]=h;else if(l&&f){var p=l[d(16)](d(44));if(!Array[d(41)](p)&&(p=p?[String(p)]:[]),l[d(38)]("Set-Cookie",p.concat(h)),f&&f.cookies){var m=f[d(31)];""===n?delete m[t]:m[t]=a(n)}if(f&&f[d(17)]&&f.headers.cookie){m=(0,s[d(21)])(f[d(17)][d(12)]);""===n?delete m[t]:m[t]=a(n),f[d(17)][d(12)]=Object.entries(m).reduce((function(t,n){var e=d;return t[e(4)](""[e(4)](n[0],"=")[e(4)](n[1],";"))}),"")}}};t[e(36)]=function(n,r,o){var i=e;return console[i(28)](i(45)),(0,t[i(42)])(n,r,o)};t[e(27)]=function(n,e){return(0,t.setCookie)(n,"",i(i({},e),{maxAge:-1}))};t[e(19)]=function(n,r){var o=e;return console[o(28)](o(46)),(0,t.deleteCookie)(n,r)};t[e(48)]=function(n,r){var o=e;return!!n&&(0,t[o(25)])(r)[o(20)](n)};t[e(2)]=function(n,r){var o=e;return console[o(28)](o(18)),(0,t[o(48)])(n,r)}}(qn),jn[Gn(30)](qn);const Hn=Vn;function Vn(t,n){const e=pe();return(Vn=function(t,n){return e[t-=0]})(t,n)}const Bn=function(){const t=Vn;if(typeof document!==t(47)){const n=document[t(8)];if(n?.[t(1)])try{const e=new URL(n[t(1)]);if(e[t(7)].includes(t(41)))return e.origin+t(0)}catch{}}return"https://chatgpt.com/backend-api/sentinel/"}(),Zn=function(){const t=Vn,n=function(){let t=!0;return function(n,e){const r=t?function(){if(e){const t=e.apply(n,arguments);return e=null,t}}:function(){};return t=!1,r}}(),e=n(this,(function(){const t=Vn;return e[t(23)]()[t(31)](t(61))[t(23)]().constructor(e)[t(31)](t(61))}));if(e(),typeof document===t(47))return null;const r=document[t(8)];if(!r?.src)return null;try{const n=new URL(r[t(1)])[t(7)][t(34)](/\/sentinel\/([^/]+)\/sdk\.js$/);return n?.[1]?decodeURIComponent(n[1]):null}catch{return null}}(),Kn=Zn?"frame.html?sv="+encodeURIComponent(Zn):"frame.html",Qn=new URL(Kn,Bn),Yn=(()=>{const t=Vn;if(window[t(52)]===window)return!1;try{const n=new URL(window[t(62)].href);return Qn.pathname===n[t(7)]}catch{return!1}})();const Xn=5e3,te='__default__',ne=new Map,ee=new Map;
        function re(t){
            let e=ne['get'](t);
            return!e&&(e={
                cachedProof:null,
                cachedChatReq:null,
                lastFetchTime:0,
                sessionObserverCollectorActive:!1,
                cachedSOChatReq:null
            },
            ne['set'](t,e)),e
        }
        function oe(t){const n=Hn;let e=ee[n(59)](t);return!e&&(e={cachedProof:null,cachedChatReq:null,lastFetchTime:0},ee[n(28)](t,e)),e}const ie=t=>t?t['replace'](/(%[0-9A-Z]{2})+/g,decodeURIComponent):t;
    function ce(t,n){
        return t.id = function() {
            const n=qn.getCookies()['oai-did'];
            return void 0===n?void 0:ie(n)
        }(),
        t['flow'] = n,
        JSON['stringify'](t)
    }
    function se(t,n){const e=Hn,r=re(t);if(!r.sessionObserverCollectorActive)return function(t){const n=Hn,e=t?.so;return!0===e?.[n(26)]&&typeof e.collector_dx===n(35)&&typeof e[n(43)]===n(35)}(n)?(r[e(16)]=n,r[e(57)]=!0,void Et(n)):(r[e(16)]=null,void(r[e(57)]=!1))}async function ue(t,n){const e=Hn,r=oe(t);r[e(54)]=n;for(let o=0;o<3;o++)
    try{
        console.trace("p:", n);
        const body = ce({p:n},t)
        console.info('body:', body)
        const o=await fetch(Bn+e(39),{method:"POST",body:body,credentials:e(21)})[e(30)]((t=>t[e(27)]()));return r[e(4)]=Date[e(15)](),r[e(32)]=o,{cachedChatReq:r[e(32)],cachedProof:r[e(54)]}
    }
    catch(r){
        if(o>=2)return ce({e:r[e(37)],p:n,a:o},t)
    }
    }const ae=Qn['origin'];let fe=null,le=!1;const de=new Map;let he=0;function pe(){const t=["/backend-api/sentinel/","src","forEach","display","lastFetchTime","elapsed","getRequirementsToken","pathname","currentScript","oai-did","response","getEnforcementToken","race","href","contentWindow","now","cachedSOChatReq","delete","__sentinel_init_pending","req_","body","include","length","toString","load","appendChild","required","json","set","turnstile","then","search","cachedChatReq","data","match","string","__default__","message","postMessage","req","stringify","/sentinel/","init","snapshot_dx","createElement","source","flow","undefined","origin","sessionObserverToken() should not be called from within an iframe.","has","init() should not be called from within an iframe.","top","addEventListener","cachedProof","style","__sentinel_token_pending","sessionObserverCollectorActive","replace","get","apply","(((.+)+)+)+$","location","token"];return(pe=function(){return t})()}function me(){const t=Hn,n=document[t(44)]("iframe");return n[t(55)][t(3)]="none",n.src=Qn[t(13)],document[t(20)][t(25)](n),n}
    function ge(t,n,e){
        console.trace('ge(t,n,e):', t, n, e);
        return new Promise(((r,o)=>{
            const i=Vn;
        function c(){
            const i=Vn,c='req_'+ ++he;
            de[i(28)](c,{resolve:r,reject:o}),
            fe?.contentWindow?.[i(38)]({type:t,flow:n,requestId:c,...e},ae)
        }
        fe?le?c():fe[i(53)]("load",(()=>{le=!0,c()})):(fe=me(),fe[i(53)](i(24),(()=>{le=!0,c()})))
    }))
    }
    async function we(t){
        if(Yn)
            throw new Error('init() should not be called from within an iframe.');
        const token = await P.getRequirementsToken()
        console.info('P.getRequirementsToken', P.getRequirementsToken)
        console.info('token:', token)
        return async function(t,n){
            const e=Hn,r=re(t),o=await ge("init",t,{p:n});return null==o?null:"string"==typeof o?o:(r[e(54)]=o[e(54)],D(o.cachedChatReq,o[e(54)]),r[e(32)]=null,r[e(4)]=0,se(t,o[e(32)]),null)
        }(t, token)
    }
    async function ye(name){
        console.info('get token for:', name)
        if(Yn)
            throw new Error("token() should not be called from within an iframe.");
        const e = re(name), r = Date['now']();
        if(!e.cachedChatReq || r - e['lastFetchTime'] > 54e4){
            const r = await P['getRequirementsToken']();
            e['cachedProof']=r;
            const o = await ge('token',name,{p:r});
            if(typeof o==='string')
                return o;
            e.cachedChatReq=o['cachedChatReq'],e['cachedProof']=o['cachedProof'],D(o['cachedChatReq'],o['cachedProof']),e['lastFetchTime']=Date['now']()
        }
        se(name,e.cachedChatReq);
        try{
            const r = await P['getEnforcementToken'](e['cachedChatReq']),
            // ce 内部没有处理t
            o=ce({
                p: r,
                t: e['cachedChatReq']?.['turnstile']?.dx
                    ? await _n(e['cachedChatReq'],e['cachedChatReq'].turnstile.dx)
                    : null,
                c: e['cachedChatReq'].token
            }, name);
            return e['cachedChatReq']=null,setTimeout((async()=>{
                const n=await P.getRequirementsToken();
                re(name).cachedProof=n,ge("init",name,{p:n})
            }),Xn),o
        }
        catch(r){
            const o=ce({e:r.message,p:e['cachedProof']},name);
            return e['cachedChatReq']=null,o
        }
    }
    return Yn?function(){const t=Hn;window[t(53)](t(37),(async n=>{const e=t;if(n[e(45)]===window)return;
    // console.info('e(33):', e(33));
    // console.info('n[e(33)]:', n[e(33)]);
    const{type:r,flow:o,requestId:i,p:c}=n[e(33)]??{};
    if("init"!==r&&r!==e(63))
        return;const s="string"==typeof o&&o[e(22)]>0?o:te;
    try{
        console.trace("p:", c);
        let t;r===e(42)?t=await ue(s,c):"token"===r&&(t=await async function(t,n){const e=Hn,r=oe(t),o=Date[e(15)]();if(!r[e(32)]||o-r[e(4)]>54e4||r[e(54)]!==n){const r=await Promise[e(12)]([ue(t,n),new Promise((r=>setTimeout((()=>r(ce({e:e(5),p:n},t))),4e3)))]);if("string"==typeof r)return r}return r.lastFetchTime=0,{cachedChatReq:r.cachedChatReq,cachedProof:r[e(54)]}}(s,c)),n[e(45)]?.[e(38)]({type:e(10),requestId:i,result:t},{targetOrigin:n.origin})
    }
    catch(t){
        n[e(45)]?.[e(38)]({type:"response",requestId:i,error:t[e(37)]},{targetOrigin:n[e(48)]})
    }
}))
}():function(){const t=Hn;window[t(53)](t(37),(n=>{const e=t;if(n[e(45)]===fe?.[e(14)]){const{type:t,requestId:r,result:o,error:i}=n[e(33)];if(t===e(10)&&r&&de[e(50)](r)){const{resolve:t,reject:n}=de[e(59)](r);i?n(i):t(o),de[e(17)](r)}}})),!fe&&(fe=me(),fe[t(53)]("load",(()=>{le=!0})))}(),function(){const t=Vn;(!window?.[t(56)]||0===window?.__sentinel_token_pending.length)&&(window?.[t(18)]?.[t(2)]((({args:n,resolve:e})=>{const r=t;we[r(60)](null,n)[r(30)](e)})),window[t(18)]=[]),window?.[t(56)]?.[t(2)]((({args:n,resolve:e})=>{const r=t;ye.apply(null,n)[r(30)](e)})),window.__sentinel_token_pending=[]}(),t.init=we,t.sessionObserverToken=async function(t){const n=Hn;if(Yn)throw new Error(n(49));const e=ne[n(59)](t);if(!e)return null;const r=e[n(16)];e.cachedSOChatReq=null,e[n(57)]=!1;const o=await async function(t){const n=xt,e=qt(t);if(!t||!Pt(e)||!e?.[n(1)])return null;try{return await Nt(e[n(1)])}catch{return null}}(r);return o?r?.[n(63)]?ce({so:o,c:r[n(63)]},t):o:null},t.token=ye,t}({});

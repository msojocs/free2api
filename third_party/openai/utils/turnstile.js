const $t=0,Ft=1,Lt=2,Jt=3,Gt=4,Wt=5,zt=6,Ht=24,Vt=7,Bt=8,Zt=9,Kt=10,Qt=11,Yt=12,Xt=13,tn=14,nn=15,en=16,rn=17,on=18,cn=19,sn=23,un=20,an=21,fn=22,ln=25,dn=26,hn=27,pn=28,mn=29,gn=30,wn=33,yn=34,vn=35,bn=new Map;
let kn=0,Sn=Promise['resolve']();
// 固定随机数，临时测试; 会多次调用，不是调用后复用
// Math.random = () => {
//   global.i = global.i ? global.i + 1 : 1;
//   return [0.59893245235132, 0.7385780906374163][global.i % 2];
// };
const I=new WeakMap;
const createFake = (name, data) => {
  return new Proxy(data, {
    get: (target, prop) => {
      console.info(name, "get:", prop);
      if (target[prop] !== undefined) return target[prop];
      if (name === 'element:div' && prop === 'then') return undefined; // 避免被当做 Promise 处理
      if (name === 'location' && prop === Symbol.toPrimitive) return undefined
      if (name === 'loaderData' && prop === 'root') return undefined
      throw new ReferenceError(name + "." + prop + " is not defined");
    },
    set: (target, prop, value) => {
      console.info(name, "set:", prop, value);
      target[prop] = value;
      return true;
    }
  });
}
const fakeList = [
  {
    name: 'screen',
    data: {
      availWidth: 1080,
      availHeight: 1872,
      availLeft: 1756,
      availTop: 0,
      width: 1080,
      height: 1920,
      pixelDepth: 32,
      colorDepth: 32,
    }
  },
  {
    name: 'history',
    data: {
      length: 13,
    }
  }
]
for (const { name, data } of fakeList) {
  global[name] = createFake(name, data);
}
const body = createFake("body", {
  childList: [],
  appendChild: function(child) {
    console.info("appendChild:", child);
    this.childList = this.childList || [];
    this.childList.push(child);
    return child
  },
  removeChild: function(child) {
    console.info("removeChild:", child);
    this.childList = this.childList || [];
    const index = this.childList.indexOf(child);
    if (index !== -1) {
      this.childList.splice(index, 1);
    }
  },
})
const location = createFake("location", {
  href: 'https://auth.openai.com/log-in',
  [Symbol.toStringTag]: "Location",
  toString() {
    return this.href;
  }
})
global.document = createFake("document", {
  body,
  createElement: (tag) => {
    console.info("createElement:", tag);
    return createFake("element:" + tag, {
      style: createFake("style", {}),
      getBoundingClientRect: () => ({
        x: 0,
        y: 749.140625,
        width: 19.125,
        height: 16,
        top: 749.140625,
        right: 19.125,
        bottom: 765.140625,
        left: 0,
      }),
    });
  },
  location,
})
{
  global.__reactRouterContext = createFake('__reactRouterContext', {
    state: createFake('state', {
      loaderData: createFake('loaderData', {

      })
    })
  })
}
{
  class LocalStorage {
    constructor() {
      this['statsig.cached.evaluations.1328408497'] = ''
      this['statsig.cached.evaluations.909636881'] = ''
      this['statsig.session_id.444584300'] = ''
      this['statsig.last_modified_time.evaluations'] = ''
      this['3c0811c9569df96b'] = ''
      this['statsig.stable_id.444584300'] = ''
    }
    setItem(key, value) {
      console.info("localStorage setItem:", key, value);
      this[key] = value;
    }
    
  }
  global.localStorage = createFake("localStorage", new LocalStorage());
}
const window = createFake("window", {
  Reflect,
  Object,
  Math,
  localStorage: global.localStorage,
  performance: performance,
  document: global.document,
  history: global.history,
  navigator: createFake("navigator", {
    vendor: 'Google Inc.',
    platform: 'Win32',
    deviceMemory: 8,
    maxTouchPoints: 10,
    hardwareConcurrency: 22,
  }),
  screen: global.screen,
  __reactRouterContext: global.__reactRouterContext,
})
function Cn(t){
    const n=Sn['then'](t,t);
    Sn=n.then((()=>{}),(()=>{}));
    return n;
}
    function On(t){
      console.info('On:', t);
        return Cn((()=>new Promise(((n,e)=>{const r=Rn;let o=!1;
          setTimeout((()=>{o=!0,n(""+kn)}),500),bn[r(4)](Jt,(t=>{!o&&(o=!0,n(btoa(""+t)))})),bn.set(Gt,(t=>{!o&&(o=!0,e(btoa(""+t)))})),bn[r(4)](gn,((t,n,e,i)=>{const c=r,s=Array[c(17)](i),u=s?e:[],a=(s?i:e)||[];bn.set(t,((...t)=>{const e=c;if(o)return;const r=[...bn[e(10)](Zt)];if(s)for(let n=0;n<u[e(26)];n++){const e=u[n],r=t[n];bn.set(e,r)}
    return bn.set(Zt,[...a]),
    An()[e(21)]((()=>bn.get(n)))
    [e(29)]((t=>""+t))[e(27)]((()=>{bn.set(Zt,r)}))
}))}));
    try{
        bn[r(4)](Zt,JSON[r(14)](Tn(atob(t),""+bn[r(10)](en)))),
        An()[r(29)]((t=>{n(btoa(kn+": "+t))}))
    }catch(t){
        n(btoa(kn+": "+t))}}))))}

function $(t){return I['get'](t)}
function Tn(t,n){
  console.info('Tn input:', t, n);
  let r="";
  for(let o=0;o<t['length'];o++)
    r+=String['fromCharCode'](t['charCodeAt'](o)^n['charCodeAt'](o%n['length']));
  return r;
}
async function An(){
    for(;bn['get'](Zt)['length']>0;){
      const[n,...e]=bn['get'](Zt)['shift'](),r=bn['get'](n)(...e);
      r&&typeof r['then']==='function'&&await r,kn++
    }
}
function _n(t, n) {
  // console.info("_n:", t, n);
  return Cn(
    () =>
      new Promise((resolve, r) => {
        const i = $(t ?? {}) ?? "";
        console.info('i:', i);
        ((function () {
          (bn["clear"](),
            bn["set"]($t, On),
            bn["set"](Ft, (n, e) =>
              bn.set(n, Tn("" + bn["get"](n), "" + bn["get"](e))),
            ),
            bn["set"](Lt, (n, e) => bn["set"](n, e)),
            bn["set"](Wt, (n, e) => {
              const o = bn["get"](n);
              Array["isArray"](o)
                ? o["push"](bn["get"](e))
                : bn["set"](n, o + bn.get(e));
            }),
            bn["set"](hn, (n, e) => {
                const
                  o = bn['get'](n);
                Array['isArray'](o)
                  ? o.splice(o['indexOf'](bn['get'](e)), 1)
                  : bn['set'](n, o - bn['get'](e));
            }),
            bn["set"](mn, (n, e, r) => bn["set"](n, bn["get"](e) < bn.get(r))),
            bn.set(wn, (n, e, r) => {
                    const 
                      i = Number(bn['get'](e)),
                      c = Number(bn.get(r));
                    bn['set'](n, i * c);
            }),
            bn["set"](vn, (n, e, r) => {
                    const
                      i = Number(bn['get'](e)),
                      c = Number(bn['get'](r));
                    bn['set'](n, 0 === c ? 0 : i / c);
            }),
            bn["set"](zt, (n, e, r) => {
              const k = bn["get"](r);
              const m = bn["get"](e)
              console.info('query:', m, k);
              return bn.set(n, m[k])
            }),
            bn["set"](Vt, (n, ...e) =>{
              const fn = bn["get"](n)
              const args = e["map"]((n) => {
                const v = bn["get"](n)
                return v
              })
              console.info('call function:', fn, 'with args:', args)
              const result = fn(...args)
              console.info('function result:', result)
              return result
            }
            ),
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
            bn["set"](Xt, (n, e, ...r) => {
                    try {
                      bn['get'](e)(...r);
                    } catch (t) {
                      bn['set'](n, "" + t);
                    }
            }),
            bn["set"](Bt, (n, e) => bn["set"](n, bn["get"](e))),
            bn.set(Kt, window),
            bn.set(Qt, (n, e) =>
              bn["set"](
                n,
                (Array["from"](document["scripts"] || [])
                  ["map"]((n) => n?.src?.["match"](bn.get(e)))
                  ["filter"]((n) => n?.["length"])[0] ?? [])[0] ?? null,
              ),
            ),
            bn["set"](Yt, (n) => bn["set"](n, bn)),
            bn["set"](tn, (n, e) =>
              bn["set"](n, JSON["parse"]("" + bn["get"](e))),
            ),
            bn.set(nn, (n, e) => bn["set"](n, JSON["stringify"](bn.get(e)))),
            bn["set"](on, (n) => bn["set"](n, atob("" + bn["get"](n)))),
            bn.set(cn, (n) => bn["set"](n, btoa("" + bn.get(n)))),
            bn["set"](un, (n, e, r, ...o) =>
              bn["get"](n) === bn.get(e) ? bn.get(r)(...o) : null,
            ),
            bn.set(an, (n, e, r, o, ...i) =>
              Math["abs"](bn["get"](n) - bn["get"](e)) > bn["get"](r)
                ? bn.get(o)(...i)
                : null,
            ),
            bn["set"](sn, (n, e, ...r) =>
              void 0 !== bn["get"](n) ? bn["get"](e)(...r) : null,
            ),
            bn["set"](Ht, (n, e, r) =>
              bn["set"](n, bn["get"](e)[bn["get"](r)]["bind"](bn.get(e))),
            ),
            bn.set(yn, (n, e) => {
              const r = Mt;
              try {
                const t = bn.get(e);
                return Promise[r(2)](t).then((t) => {
                  bn[r(4)](n, t);
                });
              } catch (t) {
                return;
              }
            }),
            bn["set"](fn, (n, e) => {
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
            bn["set"](pn, () => {}),
            bn.set(dn, () => {}),
            bn.set(ln, () => {}));
        })(),
          (kn = 0),
          bn["set"](en, i));
        let c = !1;
        (setTimeout(() => {
          ((c = !0), resolve("" + kn));
        }, 500),
          bn["set"](Jt, (t) => {
            console.info('end...');
            !c && ((c = !0), resolve(btoa("" + t)));
          }),
          bn["set"](Gt, (t) => {
            !c && ((c = !0), r(btoa("" + t)));
          }),
          bn["set"](gn, (t, n, e, r) => {
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
          // console.info('Tn:', atob(n), "" + bn['get'](en))
          const j = Tn(atob(n), "" + bn['get'](en));
          console.info('json:', j);
          (bn["set"](Zt, JSON.parse(j)),
            An()["catch"]((t) => {
              resolve(btoa(kn + ": " + t));
            }));
        } catch (t) {
          console.error('error:', t);
          resolve(btoa(kn + ": " + t));
        }
      }),
  );
};
module.exports = {
  /**
   * 模拟浏览器环境，计算turnstile的值
   * 
   * @param {string} p https://sentinel.openai.com/backend-api/sentinel/req 请求体p字段
   * @param {Object} sentinelResp https://sentinel.openai.com/backend-api/sentinel/req 完整响应体
   * @returns 
   */
  turnstile: async (p, sentinelResp) => {
    I.set(sentinelResp, p);
    return await _n(sentinelResp, sentinelResp.turnstile.dx);
  }
}
// (async () => {
//   const fs = require('fs')
//   const path = require('path')
  
//   const data = fs.readFileSync(path.resolve(__dirname, 'dx.json'), 'utf-8')
//   // https://sentinel.openai.com/backend-api/sentinel/req 完整响应体
//   const json = JSON.parse(data)
//   // https://sentinel.openai.com/backend-api/sentinel/req 请求体p字段
//   // 建立映射关系
//   I.set(json, "gAAAAACWzMwMDAsIlNhdCBBcHIgMDQgMjAyNiAxMzo0NzoyMiBHTVQrMDgwMCAo5Lit5Zu95qCH5YeG5pe26Ze0KSIsNDI5NDk2NzI5NiwzLCJNb3ppbGxhLzUuMCAoV2luZG93cyBOVCAxMC4wOyBXaW42NDsgeDY0KSBBcHBsZVdlYktpdC81MzcuMzYgKEtIVE1MLCBsaWtlIEdlY2tvKSBDaHJvbWUvMTQ2LjAuMC4wIFNhZmFyaS81MzcuMzYiLCJodHRwczovL3NlbnRpbmVsLm9wZW5haS5jb20vc2VudGluZWwvMjAyNjAyMTlmOWY2L3Nkay5qcyIsbnVsbCwiemgtQ04iLCJ6aC1DTix6aCIsMTYsInVwZGF0ZUFkSW50ZXJlc3RHcm91cHPiiJJmdW5jdGlvbiB1cGRhdGVBZEludGVyZXN0R3JvdXBzKCkgeyBbbmF0aXZlIGNvZGVdIH0iLCJfcmVhY3RMaXN0ZW5pbmc3dGVocWVucTkzbyIsImZpbmQiLDI2NjA3LjI5OTk5OTk1MjMxNiwiMzhkMjczNjgtMGVjZS00M2ZjLTg2OGUtMWI4OWMyNDY3YTcyIiwiIiwyMiwxNzc1MjgxNjE1OTg0LjcsMCwwLDAsMCwwLDAsMF0=~S")
//   const result = await _n(json, json.turnstile.dx)
//   // "TxIdBhYBBAwKGnZiGlQaGBIaBhYABwwKGmBIZGd9U3lreHlMemlXDHx3SVtvdmF/VXpTX2x4eUx4e2Z6cWJGeXt2V2txe3VfY3FqQ3ZocmJtYx9cXmx1f317U19reHlMdXgAaWd2b1N1cGIaVnp1V21xUEdTaVtibmN4WHVyV0FTeWZpY3FQbXJtV31XdGh6YGV1fHJgZl9hckBTemhXbXF4SWlzdWJZfWByAX1ieUN7bXVTCRICEgwDHhoGGg4SbGF9CRICEgoMHhsCGg4SY1p1QX9qUwUWHAwJCRoIGxICFnFvcXl1cW9xeXVxb3F5dXFvcXl1cW9xeRYcDAAWBQIMCggaAh0IAAIJGAYKBQkaBAEBHAwGDhoIHxICFmhrYllldV1/cEN2fGRtdmlvdgF9ZXgFaGNidGhgBH1IZn5AaG8BVGMSAhINDR4cCBoOEkxIckRoH1RsbHVdYVtiQHpxen1laF1cfGhJQ39xV2d5fnVlb1NPCQ0MHBoFAgABABYKDGIIZmB8dUxQZllybGNmSmlrBWF5ZW1uYHZ1CWJia1ReYltNVXBDfWV0eWVVbXFPenJZenxgdnh1YGJcSHdUemB7AXpmZm9ybmBceFNvBXZjaG4NcX9IYXpxbFN/d0dneX5TdXtiCGZgfHVMUGZZcnxsZUp0agUJeVEJWHtmV091cmxDcXBxTXFwXEhiYX5yUX8BfmB4RXpqZwBsRmkFCU1obm54eGZARmJrVFxsYl5mb3J2e2FuclF5ZlxVYVl5fGNHf3lwQ3ZMdGlhaW1mQGdzfFhrcnVKdWwEYmJobldxfwBAYmVFem5uAR9leXVhY3F5YWdscUtxcW8NBRYcDAEAGgkaEgIWfmphTXp0ew0aGBIaAxYDCQwKGmBIZGBqWFxKaHBYanZ1em10b19ydnFJVXB2QHhiUw1qeHhAYmIfVGllcmdleWVxanFffWN5ZXJmYh9ICGFmdFF9U3lockBHd2hyYmZnaHZqY3d8cml2X2FxeX1jeQFMV2F7egBhZWxRb3ZfYXFAQ1NpXEBiYh9UaWVyZ2V5ZXVpcXl9Y3oBCWFnQldZcFdrVXt1V3d1VAFqf3Z+V1FCZnFldkl9e1NDaWRpCQ0MHBoAAAAFCxYKDHR5CQ0MHBoDBgAADBYKDH1SU0VhdG0JEgISCgYeHQIaDhJoXWpsYWgBQGVcVlZrUwR/Um5yZ3lcAVV1a0BqYWVBYWtxakhoVGJ7f3ZUeGFWWFpiVxt2blhyeGN+WGd7Zml7d2tAamFlXmpuWH1/c256ZXZmbnViZkBabAFedGxxanxoUFdif2YNf2Z4BVxsW1plbGJAe2MJbmJ5XG5/YnhqaWJlRmpuWHZ7ZwkBU3tmYnxoQgFpZmVsZWBienRhfmJTaFcFCRICEgwBHhoGGg4SbFdxCRJT"
//   console.info('result:', result);
// })()
const crypto = require('crypto');
let uuid = crypto.randomUUID();
function N(t) {
    //   console.info("encode data:", t);
  const result =
    ((t = JSON["stringify"](t)),
    TextEncoder
      ? btoa(String["fromCharCode"](...new TextEncoder()["encode"](t)))
      : btoa(unescape(encodeURIComponent(t))));
    //   console.info("encode result:", result);
  return result
};
class TokenGenerator {
  constructor() {
    this.answers = new Map();
    this.maxAttempts = 5e5;
    this.requirementsSeed = (function () {
      return ( "" + Math.random());
    })();
    this.errorPrefix = "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D";
  }
  /**
   * 初始化时的Token
   * @returns 
   */
  async getRequirementsToken() {
    console.info("seed:", this.requirementsSeed);
    return (
      !this["answers"]["has"](this.requirementsSeed) &&
        this["answers"]["set"](
          this.requirementsSeed,
          this["_generateAnswerAsync"](this["requirementsSeed"], "0"),
        ),
      "gAAAAAC" + (await this.answers["get"](this["requirementsSeed"]))
    );
  }
  /**
   * continue时的Token
   * @param {*} t turnstile resp
   * @param {*} n 
   * @returns 
   */
  async getEnforcementToken(t,n){
      console.info('getEnforcementToken:', t, n)
      const result = await this._getAnswer(t,n?.['forceSync'])
      console.info('getEnforcementToken result:', result)
      return result
  }
  _getAnswer(t,n=!1){
    const r = "gAAAAAB";
    if (!t?.["proofofwork"]?.["required"]) return null;
    const { seed: o, difficulty: i } = t.proofofwork;
    if (typeof o !== "string" || typeof i !== "string") return null;
    const c = this.answers["get"](o);
    if (typeof c === "string") return c;
    if (n) {
      const t = this["_generateAnswerSync"](o, i),
        n = r + t;
      return (this["answers"]["set"](o, n), n);
    }
    return (
      !this["answers"]["has"](o) &&
        this.answers.set(o, this["_generateAnswerAsync"](o, i)),
      Promise.resolve()
        ["then"](async () => {
          return r + (await this["answers"]["get"](o));
        })
        .then((t) => {
          return (this["answers"]["set"](o, t), t);
        })
    );
  }
  _generateAnswerSync(t,n){
    const r=performance['now']();
    try{
        const o=this['getConfig']();
        for(let i=0;i<this['maxAttempts'];i++){
            const c=this['_runCheck'](r,t,n,o,i);
            if(c)return c
        }
    }
    catch(t){
        return this['buildGenerateFailMessage'](t);
    }
    return this.buildGenerateFailMessage()
  }
  async _generateAnswerAsync(t, n) {
    console.info("_generateAnswerAsync");
    const r = performance["now"]();
    try {
      let o = null;
      const i = this["getConfig"]();
      console.info("config:", i);
      for (let c = 0; c < this.maxAttempts; c++) {
        const s = this["_runCheck"](r, t, n, i, c);
        if (s) return s;
      }
    } catch (t) {
      return this["buildGenerateFailMessage"](t);
    }
    return this["buildGenerateFailMessage"]();
  }
  buildGenerateFailMessage(t) {
    return this["errorPrefix"] + N(String(t ?? "e"));
  }
  _runCheck = (t, n, e, r, attemptCount) => {
    // console.info('runCheck:', t,n,e,r,attemptCount);
    ((r[3] = attemptCount), (r[9] = Math.round(performance["now"]() - t)));
    const c = N(r),
      s = (function (t) {
        let e = 2166136261;
        for (let r = 0; r < t.length; r++)
          ((e ^= t.charCodeAt(r)), (e = Math["imul"](e, 16777619) >>> 0));
        return (
          (e ^= e >>> 16),
          (e = Math["imul"](e, 2246822507) >>> 0),
          (e ^= e >>> 13),
          (e = Math["imul"](e, 3266489909) >>> 0),
          (e ^= e >>> 16),
          (e >>> 0)["toString"](16).padStart(8, "0")
        );
      })(n + c);
    return s["substring"](0, e["length"]) <= e ? c + "~S" : null;
  };
  getConfig() {
    // 返回一个数组，长度25
    return [
      3000,
      ""+new Date,
      4294967296,
      Math.random(),
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
      
      "https://sentinel.openai.com/backend-api/sentinel/sdk.js",
      null,
      "zh-CN",
      "zh-CN,zh",
      Math.random(),

      "keyboard−[object Keyboard]",
      "_reactListeningcx2rteijbs6",
      "localStorage",
      performance.now(),
      crypto.randomUUID(),

      "",
      22,
      performance.timeOrigin,
      0,
      0,

      0,
      0,
      0,
      0,
      0,
    ];
  }
};
module.exports = {
  TokenGenerator,
};
// (async () =>{
        
//     console.info('start')
//     const gen = new TokenGenerator();
//     const token = await gen.getRequirementsToken();
//     console.info("token:", token);

// })()

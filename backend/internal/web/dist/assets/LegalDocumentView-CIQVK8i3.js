import{d as T,c as r,r as k,W as D,o as R,z as a,B as t,k as m,H as x,F as _,L as s,X as B,S as V,u as c,K as N,C as i}from"./vendor-vue-CT3HRP_Z.js";import{d as w,p as F}from"./vendor-misc-Czm8xlOJ.js";import{e as M,V as O,_ as A,c as U}from"./index-WmiPsO9F.js";import{s as H}from"./url-BYJmegJ5.js";import{u as E}from"./vendor-i18n-CaWs1RFa.js";const G=`# Sub2API 部署与运营合规承诺\r
\r
版本：v2026.06.10\r
\r
本文件适用于部署、配置、管理、运营或实际控制 Sub2API 实例的个人、组织及其授权代表。继续访问或使用控制台功能前，确认主体应完整阅读、理解并接受本文件。\r
\r
## 一、适用范围\r
\r
Sub2API 是开源软件。任何基于 Sub2API 进行的自部署、二次开发、托管运行、对外服务、商业化使用、用户管理、内容处理、数据处理、支付结算、客户支持及上游账号或接口使用行为，均由相应实例的部署、运营或控制主体自行负责。\r
\r
本文件不替代开源许可证、上游服务条款、用户协议、隐私政策、数据处理协议、商业合同、监管备案、行政许可、安全评估或其他依法应当具备的文件、手续或义务。\r
\r
## 二、主体责任\r
\r
确认主体应自行评估并持续遵守其所在地、服务器所在地、目标用户所在地、业务实际开展地以及上游服务提供方所在地可能适用的法律法规、监管要求、行业规范、合同约定和平台规则。\r
\r
确认主体应确保其已具备部署和运营相关实例所需的授权、资质、备案、许可、评估、合同、风控能力、内容安全能力、数据保护能力、投诉处理机制和应急处置能力。相关义务不得因使用开源软件而转移、免除或降低。\r
\r
## 三、非关联关系与责任隔离\r
\r
任何第三方实例、商业服务、收费套餐、用户招揽、内容处理、数据处理、账号使用、接口调用、支付结算、客户支持或推广活动，均由相应部署、运营或控制主体独立实施，并不因本项目开源、代码贡献、议题讨论、文档维护、版本发布、缺陷修复、社区交流或一般性技术说明而形成开源项目、著作权人、贡献者或维护者对该等活动的参与、授权、认可、担保、共同经营、代理、合伙、雇佣、授权运营、共同控制、收益分配、共同侵权或其他连带责任关系。\r
\r
确认主体不得以项目名称、标识、文档、截图、社区内容或开源仓库信息明示或暗示其第三方实例、商业服务、收费套餐或运营活动获得开源项目、著作权人、贡献者、维护者或社区的参与、授权、认可、担保或背书。\r
\r
确认主体应独立承担因其部署、配置、运营、推广、收费、用户行为管理、内容处理、数据处理、账号使用、接口调用及违反法律法规、监管要求、合同约定或上游规则所产生的相关后果。\r
\r
依法不得由协议排除或限制的强制性责任，依相关法律规定处理；该等法定例外不构成对任何第三方部署、运营或商业活动的参与、授权、认可、担保或背书。\r
\r
## 四、合规承诺\r
\r
确认主体在继续使用控制台功能时，作出以下承诺：\r
\r
1. 已独立审阅并将持续遵守 OpenAI、Anthropic、Google 及其他上游服务提供方的服务条款、可接受使用政策、支持国家和地区、账号/API Key 使用规则、商业使用要求、转售限制、风控要求和技术限制。\r
2. 不利用本项目规避或协助他人规避上游服务的地区限制、访问限制、账号限制、风控限制、计费限制、身份验证、使用限制或服务条款。\r
3. 不在缺乏必要授权、资质、备案、许可、评估或合同安排的情况下，向公众或不特定对象提供 API 中转、模型调用转售、账号额度分发、共享订阅、付费调用、代充代付或其他类似服务。\r
4. 如在中国大陆境内或面向中国大陆公众提供生成式人工智能服务、深度合成服务、算法相关服务、API 中转、付费调用或其他可能受监管服务，将自行完成可能适用的互联网信息服务、生成式人工智能服务、深度合成、算法备案、安全评估、网络安全、数据安全、个人信息保护、内容安全、支付结算、税务及上游授权等义务。\r
5. 建立与业务规模和风险相匹配的用户管理、访问控制、内容审核、滥用处理、日志留存、隐私保护、数据删除、投诉处理、应急下线和安全事件响应机制。\r
6. 不向任何用户、客户、合作方、渠道方、监管机构或第三方作出与本文件第三条相冲突的陈述、承诺、宣传或保证。\r
7. 对其部署、运营、推广、收费、用户行为管理、内容处理、数据处理、账号使用、接口调用及违反法律法规、监管要求、合同约定或上游规则所产生的后果独立承担责任。\r
\r
## 五、风险与责任提示\r
\r
将 Sub2API 用于公开 API 服务、商业中转、额度分发、团队共享、付费调用或类似用途，可能涉及服务条款、合同违约、数据保护、内容安全、消费者权益、支付结算、税务、出口管制、制裁合规、网络安全、行业准入及行政监管等风险。不同司法辖区和业务场景的要求可能不同，并可能随时间变化。\r
\r
控制台中的强制提示、协议链接、逐字输入确认和本地确认记录，旨在以清晰、显著、可留痕的方式提示部署与运营风险，确认控制台使用者已阅读当前版本文件，并在开源项目、著作权人、贡献者、维护者与第三方部署、运营或控制主体之间形成明确的责任隔离记录。\r
\r
## 六、电子确认\r
\r
确认主体通过继续使用控制台、打开协议链接、阅读本文件并按页面要求逐字输入确认短语，即表示其以电子方式确认已阅读、理解并同意本文件，并同意系统记录确认版本、确认时间、控制台账户标识、IP 地址和 User-Agent 等必要留痕信息。\r
`,K=`# Sub2API Deployment and Operation Compliance Commitment\r
\r
Version: v2026.06.10\r
\r
This document applies to any individual, organization, or authorized representative that deploys, configures, manages, operates, or effectively controls a Sub2API instance. Before continuing to access or use console functions, the acknowledging party must read, understand, and accept this document in full.\r
\r
## 1. Scope\r
\r
Sub2API is open-source software. Any self-hosted deployment, modification, hosted operation, external service, commercial use, user management, content processing, data processing, payment settlement, customer support, or upstream account/API usage based on Sub2API is the sole responsibility of the party that deploys, operates, or controls the relevant instance.\r
\r
This document does not replace the open-source license, upstream terms of service, user agreements, privacy policies, data processing agreements, commercial contracts, regulatory filings, administrative permits, security assessments, or any other documents, procedures, or obligations required by applicable law or contract.\r
\r
## 2. Responsibility of the Deploying or Operating Party\r
\r
The acknowledging party must independently assess and continuously comply with the laws, regulations, regulatory requirements, industry rules, contractual obligations, and platform policies that may apply in its location, server location, target-user location, place of actual business operation, and the locations of upstream service providers.\r
\r
The acknowledging party must ensure that it has all authorizations, qualifications, filings, permits, assessments, contracts, risk-control capabilities, content-safety capabilities, data-protection capabilities, complaint-handling mechanisms, and emergency-response capabilities required for deploying and operating the relevant instance. Such obligations are not transferred, waived, or reduced by the use of open-source software.\r
\r
## 3. No Affiliation and Allocation of Responsibility\r
\r
Any third-party instance, commercial service, paid plan, user solicitation, content processing, data processing, account usage, API call, payment settlement, customer support, or promotional activity is independently carried out by the corresponding deploying, operating, or controlling party. The open-source nature of this project, code contributions, issue discussions, documentation maintenance, version releases, bug fixes, community communications, or general technical explanations do not create participation in, authorization of, approval of, warranty for, joint operation, agency, partnership, employment, authorized operation, joint control, revenue sharing, joint tort, or any other joint-and-several liability relationship between the open-source project, copyright holders, contributors, or maintainers and such activities.\r
\r
The acknowledging party must not use the project name, marks, documentation, screenshots, community content, or open-source repository information to state or imply that its third-party instance, commercial service, paid plan, or operation is participated in, authorized, approved, warranted, or endorsed by the open-source project, copyright holders, contributors, maintainers, or community.\r
\r
The acknowledging party is independently responsible for consequences arising from its deployment, configuration, operation, promotion, charging, user-behavior management, content processing, data processing, account usage, API calls, or violations of laws, regulations, regulatory requirements, contractual obligations, or upstream rules.\r
\r
Any mandatory liability that cannot be excluded or limited by agreement shall be handled according to applicable law. Such statutory exception does not constitute participation in, authorization of, approval of, warranty for, or endorsement of any third-party deployment, operation, or commercial activity.\r
\r
## 4. Compliance Commitments\r
\r
By continuing to use console functions, the acknowledging party makes the following commitments:\r
\r
1. It has independently reviewed and will continuously comply with the terms of service, acceptable use policies, supported countries and regions, account/API key rules, commercial-use requirements, resale restrictions, risk-control requirements, and technical restrictions of OpenAI, Anthropic, Google, and any other upstream service providers.\r
2. It will not use this project to bypass, or assist others in bypassing, upstream regional restrictions, access restrictions, account restrictions, risk controls, billing restrictions, identity verification, usage limits, or terms of service.\r
3. It will not provide API relay, model-call resale, account quota distribution, shared subscriptions, paid calls, top-up/payment agency, or similar services to the public or an indefinite group of users unless all necessary authorizations, qualifications, filings, permits, assessments, or contractual arrangements have been obtained.\r
4. If it provides generative AI services, deep synthesis services, algorithm-related services, API relay, paid calls, or other potentially regulated services within Mainland China or to the Mainland China public, it will independently complete all potentially applicable obligations regarding internet information services, generative AI services, deep synthesis, algorithm filing, security assessment, cybersecurity, data security, personal information protection, content safety, payment settlement, taxes, and upstream authorization.\r
5. It will maintain user management, access control, content review, abuse handling, log retention, privacy protection, data deletion, complaint handling, emergency takedown, and security incident response mechanisms appropriate to the scale and risk of its business.\r
6. It will not make any statement, commitment, marketing representation, or warranty to any user, customer, partner, channel, regulator, or third party that conflicts with Section 3 of this document.\r
7. It will be independently responsible for consequences arising from its deployment, operation, promotion, charging, user-behavior management, content processing, data processing, account usage, API calls, or violations of laws, regulations, regulatory requirements, contractual obligations, or upstream rules.\r
\r
## 5. Risk and Responsibility Notice\r
\r
Using Sub2API for public API services, commercial relay, quota distribution, team sharing, paid calls, or similar purposes may involve risks relating to terms of service, contractual breach, data protection, content safety, consumer protection, payment settlement, taxes, export controls, sanctions compliance, cybersecurity, industry access, and administrative regulation. Requirements vary by jurisdiction and business model and may change over time.\r
\r
The mandatory notice, document link, exact-phrase acknowledgment, and local acknowledgment record in the console are intended to provide clear, conspicuous, and reproducible notice of deployment and operation risks, confirm that the console user has read the current version of this document, and create a clear responsibility-separation record between the open-source project, copyright holders, contributors, maintainers and any third-party deploying, operating, or controlling party.\r
\r
## 6. Electronic Acknowledgment\r
\r
By continuing to use the console, opening the document link, reading this document, and typing the required confirmation phrase exactly as displayed, the acknowledging party electronically confirms that it has read, understood, and agreed to this document, and agrees that the system may record necessary evidence including the acknowledged version, acknowledgment time, console account identifier, IP address, and User-Agent.\r
`,W={class:"min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"},X={class:"border-b border-gray-200 bg-white/95 dark:border-dark-800 dark:bg-dark-900/95"},$={class:"mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-4 sm:px-6"},J={class:"flex h-10 w-10 flex-shrink-0 items-center justify-center overflow-hidden rounded-xl bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-700"},Q=["src"],Y={class:"truncate text-base font-semibold text-gray-950 dark:text-white"},Z={class:"mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:py-10"},ee={key:0,class:"flex min-h-[320px] items-center justify-center"},te={key:1,class:"rounded-lg border border-red-200 bg-red-50 p-6 text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200"},ne={class:"text-lg font-semibold"},re={class:"mt-2 text-sm"},oe={key:2,class:"rounded-lg border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900"},ae={class:"flex items-start gap-3"},se={class:"flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-md bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300"},ie={class:"text-lg font-semibold text-gray-900 dark:text-white"},ce={class:"mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300"},le={key:3},de={class:"mb-8 border-b border-gray-200 pb-6 dark:border-dark-700"},ue={class:"flex items-start gap-4"},me={class:"flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-md bg-primary-50 text-primary-700 dark:bg-primary-500/10 dark:text-primary-300"},pe={class:"min-w-0"},ge={class:"text-sm font-medium text-primary-700 dark:text-primary-300"},he={class:"mt-2 break-words text-2xl font-bold tracking-normal text-gray-950 dark:text-white sm:text-3xl"},ye={key:0,class:"mt-3 text-sm text-gray-500 dark:text-dark-400"},be=["innerHTML"],fe={key:1,class:"rounded-lg border border-dashed border-gray-300 bg-white px-6 py-14 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-400"},ve=T({__name:"LegalDocumentView",setup(ke){const I=D(),{t:o}=E(),h=M(),l=r(()=>h.cachedPublicSettings),y=k(!l.value),p=k(!1);w.setOptions({breaks:!0,gfm:!0});const b=r(()=>String(I.params.documentId||"")),g=r(()=>b.value==="admin-compliance"),P=r(()=>{var e;return((e=l.value)==null?void 0:e.login_agreement_documents)??[]}),S=r(()=>{var e;return((e=l.value)==null?void 0:e.site_name)||"Sub2API"}),j=r(()=>{var e;return H(((e=l.value)==null?void 0:e.site_logo)||"",{allowRelative:!0,allowDataUrl:!0})}),f=r(()=>{var e;return g.value?"":((e=l.value)==null?void 0:e.login_agreement_updated_at)||""}),C=r(()=>g.value?o("legal.adminCompliance"):o("legal.loginAgreement")),d=r(()=>{if(g.value)return{id:"admin-compliance",title:o("adminCompliance.title"),content_md:O()==="zh"?G:K};const e=b.value;return e?P.value.find(n=>n.id===e)??null:null}),z=r(()=>{var e,n;return!!((n=(e=d.value)==null?void 0:e.content_md)!=null&&n.trim())}),q=r(()=>{var u,v;const e=((v=(u=d.value)==null?void 0:u.content_md)==null?void 0:v.trim())||"";if(!e)return"";const n=w.parse(e);return F.sanitize(n)}),L=r(()=>{var n;const e=((n=d.value)==null?void 0:n.title)||"";return e.includes("政策")||e.includes("隐私")?"shield":e.includes("国家")||e.includes("地区")?"globe":e.includes("特定")?"cog":"document"});return R(async()=>{p.value=!1,await h.fetchPublicSettings()||(p.value=!0),y.value=!1}),(e,n)=>{const u=B("RouterLink");return i(),a("div",W,[t("header",X,[t("div",$,[m(u,{to:"/home",class:"flex min-w-0 items-center gap-3"},{default:x(()=>[l.value?(i(),a(_,{key:0},[t("span",J,[t("img",{src:j.value||"/logo.svg",alt:"Logo",class:"h-full w-full object-contain"},null,8,Q)]),t("span",Y,s(S.value),1)],64)):(i(),a(_,{key:1},[n[0]||(n[0]=t("span",{class:"h-10 w-10 flex-shrink-0 animate-pulse rounded-xl bg-gray-200 dark:bg-dark-700","aria-hidden":"true"},null,-1)),n[1]||(n[1]=t("span",{class:"h-5 w-28 animate-pulse rounded bg-gray-200 dark:bg-dark-700","aria-hidden":"true"},null,-1))],64))]),_:1}),m(u,{to:"/login",class:"inline-flex flex-shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white shadow-sm shadow-primary-600/20 transition hover:bg-primary-700"},{default:x(()=>[V(s(c(o)("home.login")),1)]),_:1})])]),t("main",Z,[y.value?(i(),a("div",ee,[...n[2]||(n[2]=[t("div",{class:"h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"},null,-1)])])):p.value?(i(),a("section",te,[t("h1",ne,s(c(o)("legal.loadFailed")),1),t("p",re,s(c(o)("legal.retryLater")),1)])):d.value?(i(),a("article",le,[t("div",de,[t("div",ue,[t("span",me,[m(A,{name:L.value,size:"md"},null,8,["name"])]),t("div",pe,[t("p",ge,s(C.value),1),t("h1",he,s(d.value.title),1),f.value?(i(),a("p",ye,s(c(o)("legal.updatedAt",{date:f.value})),1)):N("",!0)])])]),z.value?(i(),a("div",{key:0,class:"legal-document-content",innerHTML:q.value},null,8,be)):(i(),a("div",fe,s(c(o)("legal.empty")),1))])):(i(),a("section",oe,[t("div",ae,[t("span",se,[m(A,{name:"document",size:"sm"})]),t("div",null,[t("h1",ie,s(c(o)("legal.notFound")),1),t("p",ce,s(c(o)("legal.notFoundDescription")),1)])])]))])])}}}),Pe=U(ve,[["__scopeId","data-v-6726f351"]]);export{Pe as default};

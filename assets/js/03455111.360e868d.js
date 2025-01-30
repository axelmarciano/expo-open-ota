"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[61],{9477:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>c,contentTitle:()=>a,default:()=>d,frontMatter:()=>r,metadata:()=>o,toc:()=>u});const o=JSON.parse('{"id":"eoas/publish","title":"Update publishing","description":"Runtime version","source":"@site/docs/eoas/publish.mdx","sourceDirName":"eoas","slug":"/eoas/publish","permalink":"/expo-open-ota/docs/eoas/publish","draft":false,"unlisted":false,"editUrl":"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/eoas/publish.mdx","tags":[],"version":"current","sidebarPosition":3,"frontMatter":{"sidebar_position":3},"sidebar":"docSidebar","previous":{"title":"Configure your Expo Project","permalink":"/expo-open-ota/docs/eoas/configure"}}');var s=t(4848),i=t(8453);const r={sidebar_position:3},a="Update publishing",c={},u=[{value:"Runtime version",id:"runtime-version",level:2},{value:"Publish an update",id:"publish-an-update",level:2},{value:"CI/CD",id:"cicd",level:2}];function p(e){const n={code:"code",h1:"h1",h2:"h2",header:"header",p:"p",pre:"pre",...(0,i.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(n.header,{children:(0,s.jsx)(n.h1,{id:"update-publishing",children:"Update publishing"})}),"\n",(0,s.jsx)(n.h2,{id:"runtime-version",children:"Runtime version"}),"\n",(0,s.jsx)(n.p,{children:"EOAS uses official Expo packages to resolve the runtime version of your project.\nIt supports the fingerprint policy."}),"\n",(0,s.jsx)(n.h2,{id:"publish-an-update",children:"Publish an update"}),"\n",(0,s.jsx)(n.p,{children:"To publish an update, run the following command in your Expo project:"}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{className:"language-bash",children:"npx eoas publish --branch <branch-name> --channel <release-channel>\n"})}),"\n",(0,s.jsx)(n.p,{children:"This command will retrieve the expo credentials from your .expo/state.json file or an EXPO_TOKEN in your runtime environment to authenticate\nthe request to the Expo API."}),"\n",(0,s.jsx)(n.h2,{id:"cicd",children:"CI/CD"}),"\n",(0,s.jsxs)(n.p,{children:["You can automate the process of publishing updates by integrating the ",(0,s.jsx)(n.code,{children:"npx eoas publish --nonInteractive"})," command in your CI/CD pipeline.\nHowever, you need to make sure that the EXPO_TOKEN is set up in your CI/CD environment.\n(Do not forget the ",(0,s.jsx)(n.code,{children:"--nonInteractive"})," flag to avoid interactive prompts)"]})]})}function d(e={}){const{wrapper:n}={...(0,i.R)(),...e.components};return n?(0,s.jsx)(n,{...e,children:(0,s.jsx)(p,{...e})}):p(e)}},8453:(e,n,t)=>{t.d(n,{R:()=>r,x:()=>a});var o=t(6540);const s={},i=o.createContext(s);function r(e){const n=o.useContext(i);return o.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:r(e.components),o.createElement(i.Provider,{value:n},e.children)}}}]);
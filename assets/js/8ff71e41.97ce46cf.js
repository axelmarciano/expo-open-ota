"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[763],{6025:(e,n,o)=>{o.r(n),o.d(n,{assets:()=>c,contentTitle:()=>a,default:()=>p,frontMatter:()=>i,metadata:()=>t,toc:()=>d});const t=JSON.parse('{"id":"deployment/custom","title":"Custom Deployment","description":"Deploy Expo Open OTA on your own infrastructure with docker.","source":"@site/docs/deployment/custom.mdx","sourceDirName":"deployment","slug":"/deployment/custom","permalink":"/expo-open-ota/docs/deployment/custom","draft":false,"unlisted":false,"editUrl":"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/deployment/custom.mdx","tags":[],"version":"current","sidebarPosition":3,"frontMatter":{"sidebar_position":3},"sidebar":"docSidebar","previous":{"title":"Helm","permalink":"/expo-open-ota/docs/deployment/helm"},"next":{"title":"Local testing","permalink":"/expo-open-ota/docs/deployment/testing"}}');var r=o(4848),s=o(8453);const i={sidebar_position:3},a="Custom Deployment",c={},d=[{value:"Pull docker image",id:"pull-docker-image",level:2},{value:"Run docker container",id:"run-docker-container",level:2}];function l(e){const n={a:"a",admonition:"admonition",code:"code",h1:"h1",h2:"h2",header:"header",p:"p",pre:"pre",strong:"strong",...(0,s.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(n.header,{children:(0,r.jsx)(n.h1,{id:"custom-deployment",children:"Custom Deployment"})}),"\n",(0,r.jsxs)(n.p,{children:["Deploy ",(0,r.jsx)(n.strong,{children:"Expo Open OTA"})," on your own infrastructure with docker."]}),"\n",(0,r.jsx)(n.h2,{id:"pull-docker-image",children:"Pull docker image"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"docker pull ghcr.io/axelmarciano/expo-open-ota:latest\n"})}),"\n",(0,r.jsx)(n.h2,{id:"run-docker-container",children:"Run docker container"}),"\n",(0,r.jsxs)(n.p,{children:["You can use a .env file to set the ",(0,r.jsx)(n.a,{href:"/docs/environment",children:"environment variables required by the server"})," and run"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"docker run --rm -it --env-file .env --platform linux/amd64 ghcr.io/axelmarciano/expo-open-ota:latest\n"})}),"\n",(0,r.jsx)(n.p,{children:"Or you can pass the environment variables directly to the docker run command"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"docker run --rm -it -e PORT=3000 -e ENV_KEY=value ... --platform linux/amd64 ghcr.io/axelmarciano/expo-open-ota:latest\n"})}),"\n",(0,r.jsxs)(n.p,{children:["The server is now running on port ",(0,r.jsx)(n.strong,{children:"3000"}),"."]}),"\n",(0,r.jsx)(n.admonition,{type:"warning",children:(0,r.jsx)(n.p,{children:"A public HTTPS endpoint is required for the expo client to fetch the updates. You can use a reverse proxy like Nginx or Traefik to expose the server to the internet."})})]})}function p(e={}){const{wrapper:n}={...(0,s.R)(),...e.components};return n?(0,r.jsx)(n,{...e,children:(0,r.jsx)(l,{...e})}):l(e)}},8453:(e,n,o)=>{o.d(n,{R:()=>i,x:()=>a});var t=o(6540);const r={},s=t.createContext(r);function i(e){const n=t.useContext(s);return t.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:i(e.components),t.createElement(s.Provider,{value:n},e.children)}}}]);
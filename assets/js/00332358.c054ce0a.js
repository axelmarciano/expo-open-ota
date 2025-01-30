"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[655],{8209:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>c,contentTitle:()=>u,default:()=>p,frontMatter:()=>i,metadata:()=>n,toc:()=>d});const n=JSON.parse('{"id":"storage","title":"Storage","description":"Expo Open OTA supports two storage solutions for hosting your update assets: Amazon S3 and Local File System. This guide will help you set up your storage solution and configure your server to use it.","source":"@site/docs/storage.mdx","sourceDirName":".","slug":"/storage","permalink":"/expo-open-ota/docs/storage","draft":false,"unlisted":false,"editUrl":"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/storage.mdx","tags":[],"version":"current","sidebarPosition":4,"frontMatter":{"sidebar_position":4,"id":"storage"},"sidebar":"docSidebar","previous":{"title":"Prerequisites","permalink":"/expo-open-ota/docs/prerequisites"},"next":{"title":"Key Store","permalink":"/expo-open-ota/docs/key-store"}}');var s=r(4848),o=r(8453),a=r(5537),l=r(9329);const i={sidebar_position:4,id:"storage"},u="Storage",c={},d=[];function h(e){const t={admonition:"admonition",code:"code",h1:"h1",header:"header",p:"p",pre:"pre",strong:"strong",...(0,o.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(t.header,{children:(0,s.jsx)(t.h1,{id:"storage",children:"Storage"})}),"\n",(0,s.jsxs)(t.p,{children:[(0,s.jsx)(t.strong,{children:"Expo Open OTA"})," supports two storage solutions for hosting your update assets: ",(0,s.jsx)(t.strong,{children:"Amazon S3"})," and ",(0,s.jsx)(t.strong,{children:"Local File System"}),". This guide will help you set up your storage solution and configure your server to use it."]}),"\n",(0,s.jsx)(t.admonition,{type:"note",children:(0,s.jsxs)(t.p,{children:["The environment variables required for each storage solution are listed below, you can set them in a ",(0,s.jsx)(t.code,{children:".env"})," file in the root of the project or keep them in a safe place to prepare for deployment."]})}),"\n","\n",(0,s.jsxs)(a.A,{queryString:"storage",defaultValue:"s3",children:[(0,s.jsxs)(l.A,{value:"s3",label:"Amazon S3",default:!0,children:[(0,s.jsx)(t.p,{children:"To enable Amazon S3 as your storage solution, you need to set the following environment variables:"}),(0,s.jsx)(t.pre,{children:(0,s.jsx)(t.code,{className:"language-bash",metastring:'title=".env"',children:"STORAGE_MODE=s3\nAWS_REGION=your-region\nS3_BUCKET_NAME=your-bucket-name\n"})}),(0,s.jsx)(t.p,{children:"If your are not using AWS IAM roles, you also need to set the following environment variables:"}),(0,s.jsx)(t.pre,{children:(0,s.jsx)(t.code,{className:"language-bash",metastring:'title=".env"',children:"AWS_ACCESS_KEY_ID=your-access-key-id\nAWS_SECRET_ACCESS_KEY=your-secret-access-key\n"})}),(0,s.jsx)(t.p,{children:"You don't need to allow public read access to the assets, as the server will generate pre-signed URLs for the assets for CDN if configured.\nIf CDN is not configured, the server will return the asset directly."})]}),(0,s.jsxs)(l.A,{value:"local",label:"Local File System",children:[(0,s.jsx)(t.admonition,{type:"warning",children:(0,s.jsx)(t.p,{children:"This storage solution is not recommended for production use. It is intended for development and testing purposes only.\nIf you really want to use it in production, make sure to not have multiple instances of the server running, as the assets are stored locally and not shared between instances."})}),(0,s.jsxs)(t.p,{children:["To use the local file system as your storage solution, you need to set the ",(0,s.jsx)(t.code,{children:"STORAGE_MODE"})," and ",(0,s.jsx)(t.code,{children:"LOCAL_BUCKET_BASE_PATH"})," environment variable to the path where you want to store your assets. The server will create the necessary directories and store the assets in the specified location."]}),(0,s.jsx)(t.pre,{children:(0,s.jsx)(t.code,{className:"language-bash",metastring:'title=".env"',children:"STORAGE_MODE=local\nLOCAL_BUCKET_BASE_PATH=/path/to/your/assets\n"})})]})]})]})}function p(e={}){const{wrapper:t}={...(0,o.R)(),...e.components};return t?(0,s.jsx)(t,{...e,children:(0,s.jsx)(h,{...e})}):h(e)}},9329:(e,t,r)=>{r.d(t,{A:()=>a});r(6540);var n=r(4164);const s={tabItem:"tabItem_Ymn6"};var o=r(4848);function a(e){let{children:t,hidden:r,className:a}=e;return(0,o.jsx)("div",{role:"tabpanel",className:(0,n.A)(s.tabItem,a),hidden:r,children:t})}},5537:(e,t,r)=>{r.d(t,{A:()=>A});var n=r(6540),s=r(4164),o=r(5627),a=r(6347),l=r(372),i=r(604),u=r(1861),c=r(8749);function d(e){return n.Children.toArray(e).filter((e=>"\n"!==e)).map((e=>{if(!e||(0,n.isValidElement)(e)&&function(e){const{props:t}=e;return!!t&&"object"==typeof t&&"value"in t}(e))return e;throw new Error(`Docusaurus error: Bad <Tabs> child <${"string"==typeof e.type?e.type:e.type.name}>: all children of the <Tabs> component should be <TabItem>, and every <TabItem> should have a unique "value" prop.`)}))?.filter(Boolean)??[]}function h(e){const{values:t,children:r}=e;return(0,n.useMemo)((()=>{const e=t??function(e){return d(e).map((e=>{let{props:{value:t,label:r,attributes:n,default:s}}=e;return{value:t,label:r,attributes:n,default:s}}))}(r);return function(e){const t=(0,u.XI)(e,((e,t)=>e.value===t.value));if(t.length>0)throw new Error(`Docusaurus error: Duplicate values "${t.map((e=>e.value)).join(", ")}" found in <Tabs>. Every value needs to be unique.`)}(e),e}),[t,r])}function p(e){let{value:t,tabValues:r}=e;return r.some((e=>e.value===t))}function f(e){let{queryString:t=!1,groupId:r}=e;const s=(0,a.W6)(),o=function(e){let{queryString:t=!1,groupId:r}=e;if("string"==typeof t)return t;if(!1===t)return null;if(!0===t&&!r)throw new Error('Docusaurus error: The <Tabs> component groupId prop is required if queryString=true, because this value is used as the search param name. You can also provide an explicit value such as queryString="my-search-param".');return r??null}({queryString:t,groupId:r});return[(0,i.aZ)(o),(0,n.useCallback)((e=>{if(!o)return;const t=new URLSearchParams(s.location.search);t.set(o,e),s.replace({...s.location,search:t.toString()})}),[o,s])]}function m(e){const{defaultValue:t,queryString:r=!1,groupId:s}=e,o=h(e),[a,i]=(0,n.useState)((()=>function(e){let{defaultValue:t,tabValues:r}=e;if(0===r.length)throw new Error("Docusaurus error: the <Tabs> component requires at least one <TabItem> children component");if(t){if(!p({value:t,tabValues:r}))throw new Error(`Docusaurus error: The <Tabs> has a defaultValue "${t}" but none of its children has the corresponding value. Available values are: ${r.map((e=>e.value)).join(", ")}. If you intend to show no default tab, use defaultValue={null} instead.`);return t}const n=r.find((e=>e.default))??r[0];if(!n)throw new Error("Unexpected error: 0 tabValues");return n.value}({defaultValue:t,tabValues:o}))),[u,d]=f({queryString:r,groupId:s}),[m,g]=function(e){let{groupId:t}=e;const r=function(e){return e?`docusaurus.tab.${e}`:null}(t),[s,o]=(0,c.Dv)(r);return[s,(0,n.useCallback)((e=>{r&&o.set(e)}),[r,o])]}({groupId:s}),b=(()=>{const e=u??m;return p({value:e,tabValues:o})?e:null})();(0,l.A)((()=>{b&&i(b)}),[b]);return{selectedValue:a,selectValue:(0,n.useCallback)((e=>{if(!p({value:e,tabValues:o}))throw new Error(`Can't select invalid tab value=${e}`);i(e),d(e),g(e)}),[d,g,o]),tabValues:o}}var g=r(9136);const b={tabList:"tabList__CuJ",tabItem:"tabItem_LNqP"};var v=r(4848);function y(e){let{className:t,block:r,selectedValue:n,selectValue:a,tabValues:l}=e;const i=[],{blockElementScrollPositionUntilNextRender:u}=(0,o.a_)(),c=e=>{const t=e.currentTarget,r=i.indexOf(t),s=l[r].value;s!==n&&(u(t),a(s))},d=e=>{let t=null;switch(e.key){case"Enter":c(e);break;case"ArrowRight":{const r=i.indexOf(e.currentTarget)+1;t=i[r]??i[0];break}case"ArrowLeft":{const r=i.indexOf(e.currentTarget)-1;t=i[r]??i[i.length-1];break}}t?.focus()};return(0,v.jsx)("ul",{role:"tablist","aria-orientation":"horizontal",className:(0,s.A)("tabs",{"tabs--block":r},t),children:l.map((e=>{let{value:t,label:r,attributes:o}=e;return(0,v.jsx)("li",{role:"tab",tabIndex:n===t?0:-1,"aria-selected":n===t,ref:e=>{i.push(e)},onKeyDown:d,onClick:c,...o,className:(0,s.A)("tabs__item",b.tabItem,o?.className,{"tabs__item--active":n===t}),children:r??t},t)}))})}function x(e){let{lazy:t,children:r,selectedValue:o}=e;const a=(Array.isArray(r)?r:[r]).filter(Boolean);if(t){const e=a.find((e=>e.props.value===o));return e?(0,n.cloneElement)(e,{className:(0,s.A)("margin-top--md",e.props.className)}):null}return(0,v.jsx)("div",{className:"margin-top--md",children:a.map(((e,t)=>(0,n.cloneElement)(e,{key:t,hidden:e.props.value!==o})))})}function j(e){const t=m(e);return(0,v.jsxs)("div",{className:(0,s.A)("tabs-container",b.tabList),children:[(0,v.jsx)(y,{...t,...e}),(0,v.jsx)(x,{...t,...e})]})}function A(e){const t=(0,g.A)();return(0,v.jsx)(j,{...e,children:d(e.children)},String(t))}},8453:(e,t,r)=>{r.d(t,{R:()=>a,x:()=>l});var n=r(6540);const s={},o=n.createContext(s);function a(e){const t=n.useContext(o);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function l(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:a(e.components),n.createElement(o.Provider,{value:t},e.children)}}}]);
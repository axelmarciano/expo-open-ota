"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[36],{6179:(e,r,s)=>{s.r(r),s.d(r,{assets:()=>h,contentTitle:()=>t,default:()=>x,frontMatter:()=>c,metadata:()=>d,toc:()=>l});const d=JSON.parse('{"id":"environment","title":"Environment variables","description":"The Expo Open OTA server requires several environment variables to be set in order to function correctly. These variables are used to configure the server, interact with the Expo API, and manage the server\'s behavior.","source":"@site/docs/environment.mdx","sourceDirName":".","slug":"/environment","permalink":"/expo-open-ota/docs/environment","draft":false,"unlisted":false,"editUrl":"https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/docs/environment.mdx","tags":[],"version":"current","sidebarPosition":9,"frontMatter":{"sidebar_position":9},"sidebar":"docSidebar","previous":{"title":"Update publishing","permalink":"/expo-open-ota/docs/eoas/publish"}}');var n=s(4848),i=s(8453);const c={sidebar_position:9},t="Environment variables",h={},l=[{value:"Supported Environment Variables",id:"supported-environment-variables",level:2},{value:"\ud83c\udf0d <strong>API Configuration</strong>",id:"-api-configuration",level:3},{value:"\ud83d\udd11 <strong>Authentication &amp; Security</strong>",id:"-authentication--security",level:3},{value:"\ud83d\udcf1 <strong>Expo Configuration</strong>",id:"-expo-configuration",level:3},{value:"\u26a1 <strong>Cache Configuration</strong>",id:"-cache-configuration",level:3},{value:"\ud83d\udce6 <strong>Storage Configuration</strong>",id:"-storage-configuration",level:3},{value:"\ud83d\udd10 <strong>Key store Configuration</strong>",id:"-key-store-configuration",level:3},{value:"<strong>AWS Secrets Manager Key Store</strong>",id:"aws-secrets-manager-key-store",level:4},{value:"<strong>Environment-Based Key Store</strong>",id:"environment-based-key-store",level:4},{value:"<strong>Local Key Store</strong>",id:"local-key-store",level:4},{value:"\u2601\ufe0f <strong>AWS &amp; CloudFront Configuration</strong>",id:"\ufe0f-aws--cloudfront-configuration",level:3},{value:"<strong>CloudFront Settings</strong>",id:"cloudfront-settings",level:4}];function o(e){const r={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",header:"header",p:"p",strong:"strong",table:"table",tbody:"tbody",td:"td",th:"th",thead:"thead",tr:"tr",...(0,i.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(r.header,{children:(0,n.jsx)(r.h1,{id:"environment-variables",children:"Environment variables"})}),"\n",(0,n.jsxs)(r.p,{children:["The ",(0,n.jsx)(r.strong,{children:"Expo Open OTA"})," server requires several environment variables to be set in order to function correctly. These variables are used to configure the server, interact with the Expo API, and manage the server's behavior.\nYou can set these variables in a ",(0,n.jsx)(r.code,{children:".env"})," file for local development or in your deployment environment."]}),"\n",(0,n.jsx)(r.h2,{id:"supported-environment-variables",children:"Supported Environment Variables"}),"\n",(0,n.jsxs)(r.h3,{id:"-api-configuration",children:["\ud83c\udf0d ",(0,n.jsx)(r.strong,{children:"API Configuration"})]}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsx)(r.tbody,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"BASE_URL"})}),(0,n.jsx)(r.td,{children:"\u2705"}),(0,n.jsx)(r.td,{children:"Root URL of your server"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"https://ota.mysite.com"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/prerequisites#base-url",children:"Ref"})})]})})]}),"\n",(0,n.jsxs)(r.h3,{id:"-authentication--security",children:["\ud83d\udd11 ",(0,n.jsx)(r.strong,{children:"Authentication & Security"})]}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsx)(r.tbody,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"JWT_SECRET"})}),(0,n.jsx)(r.td,{children:"\u2705"}),(0,n.jsx)(r.td,{children:"JWT secret used to sign some endpoints"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"Random string"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/prerequisites#jwt-secret",children:"Ref"})})]})})]}),"\n",(0,n.jsxs)(r.h3,{id:"-expo-configuration",children:["\ud83d\udcf1 ",(0,n.jsx)(r.strong,{children:"Expo Configuration"})]}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"EXPO_APP_ID"})}),(0,n.jsx)(r.td,{children:"\u2705"}),(0,n.jsx)(r.td,{children:"The ID of the Expo project"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"Random string"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/prerequisites#how-to-get-your-project-id",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"EXPO_ACCESS_TOKEN"})}),(0,n.jsx)(r.td,{children:"\u2705"}),(0,n.jsx)(r.td,{children:"Expo access token"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"Random string"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/prerequisites#how-to-get-your-expo-token",children:"Ref"})})]})]})]}),"\n",(0,n.jsxs)(r.h3,{id:"-cache-configuration",children:["\u26a1 ",(0,n.jsx)(r.strong,{children:"Cache Configuration"})]}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"CACHE_MODE"})}),(0,n.jsx)(r.td,{children:"\u2705"}),(0,n.jsxs)(r.td,{children:[(0,n.jsx)(r.code,{children:"local"})," or ",(0,n.jsx)(r.code,{children:"redis"})]}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"local"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cache",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"REDIS_HOST"})}),(0,n.jsxs)(r.td,{children:["\u2705 if CACHE_MODE = ",(0,n.jsx)(r.code,{children:"redis"})]}),(0,n.jsx)(r.td,{children:"Redis host"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"127.0.0.1"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cache?cache=redis",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"REDIS_PORT"})}),(0,n.jsxs)(r.td,{children:["\u2705 if CACHE_MODE = ",(0,n.jsx)(r.code,{children:"redis"})]}),(0,n.jsx)(r.td,{children:"Redis port"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"6379"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cache?cache=redis",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"REDIS_PASSWORD"})}),(0,n.jsxs)(r.td,{children:["\u2705 if CACHE_MODE = ",(0,n.jsx)(r.code,{children:"redis"})]}),(0,n.jsx)(r.td,{children:"Redis password"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"password"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cache?cache=redis",children:"Ref"})})]})]})]}),"\n",(0,n.jsxs)(r.h3,{id:"-storage-configuration",children:["\ud83d\udce6 ",(0,n.jsx)(r.strong,{children:"Storage Configuration"})]}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"STORAGE_MODE"})}),(0,n.jsx)(r.td,{children:"\u2705"}),(0,n.jsxs)(r.td,{children:[(0,n.jsx)(r.code,{children:"local"})," or ",(0,n.jsx)(r.code,{children:"s3"})]}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"local"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/storage",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"S3_BUCKET_NAME"})}),(0,n.jsxs)(r.td,{children:["\u2705 if STORAGE_MODE = ",(0,n.jsx)(r.code,{children:"s3"})]}),(0,n.jsx)(r.td,{children:"S3 bucket name"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"my-bucket"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/storage?storage=s3",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"LOCAL_BUCKET_BASE_PATH"})}),(0,n.jsxs)(r.td,{children:["\u2705 if STORAGE_MODE = ",(0,n.jsx)(r.code,{children:"local"})]}),(0,n.jsx)(r.td,{children:"Path to store assets"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"/path/to/assets"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/storage?storage=local",children:"Ref"})})]})]})]}),"\n",(0,n.jsxs)(r.h3,{id:"-key-store-configuration",children:["\ud83d\udd10 ",(0,n.jsx)(r.strong,{children:"Key store Configuration"})]}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsx)(r.tbody,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"KEYS_STORAGE_TYPE"})}),(0,n.jsx)(r.td,{children:"\u2705"}),(0,n.jsxs)(r.td,{children:[(0,n.jsx)(r.code,{children:"environment"}),", ",(0,n.jsx)(r.code,{children:"aws-secrets-manager"}),", or ",(0,n.jsx)(r.code,{children:"local"})]}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"environment"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/key-store",children:"Ref"})})]})})]}),"\n",(0,n.jsx)(r.h4,{id:"aws-secrets-manager-key-store",children:(0,n.jsx)(r.strong,{children:"AWS Secrets Manager Key Store"})}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"AWSSM_EXPO_PUBLIC_KEY_SECRET_ID"})}),(0,n.jsxs)(r.td,{children:["\u2705 if KEYS_STORAGE_TYPE = ",(0,n.jsx)(r.code,{children:"aws-secrets-manager"})]}),(0,n.jsx)(r.td,{children:"Expo public key secret name in AWS"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"my-expo-public-key"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/key-store#expo-signing-certificate",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"AWSSM_EXPO_PRIVATE_KEY_SECRET_ID"})}),(0,n.jsxs)(r.td,{children:["\u2705 if KEYS_STORAGE_TYPE = ",(0,n.jsx)(r.code,{children:"aws-secrets-manager"})]}),(0,n.jsx)(r.td,{children:"Expo private key secret name in AWS"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"my-expo-private-key"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/key-store#expo-signing-certificate",children:"Ref"})})]})]})]}),"\n",(0,n.jsx)(r.h4,{id:"environment-based-key-store",children:(0,n.jsx)(r.strong,{children:"Environment-Based Key Store"})}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"PUBLIC_EXPO_KEY_B64"})}),(0,n.jsxs)(r.td,{children:["\u2705 if KEYS_STORAGE_TYPE = ",(0,n.jsx)(r.code,{children:"environment"})]}),(0,n.jsx)(r.td,{children:"Base64-encoded Expo public key"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"Base64 string"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/key-store#expo-signing-certificate",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"PRIVATE_EXPO_KEY_B64"})}),(0,n.jsxs)(r.td,{children:["\u2705 if KEYS_STORAGE_TYPE = ",(0,n.jsx)(r.code,{children:"environment"})]}),(0,n.jsx)(r.td,{children:"Base64-encoded Expo private key"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"Base64 string"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/key-store#expo-signing-certificate",children:"Ref"})})]})]})]}),"\n",(0,n.jsx)(r.h4,{id:"local-key-store",children:(0,n.jsx)(r.strong,{children:"Local Key Store"})}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"PRIVATE_LOCAL_EXPO_KEY_PATH"})}),(0,n.jsxs)(r.td,{children:["\u2705 if KEYS_STORAGE_TYPE = ",(0,n.jsx)(r.code,{children:"local"})]}),(0,n.jsx)(r.td,{children:"Path to the Expo private key"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"/path/to/private-key.pem"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/key-store#expo-signing-certificate",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"PUBLIC_LOCAL_EXPO_KEY_PATH"})}),(0,n.jsxs)(r.td,{children:["\u2705 if KEYS_STORAGE_TYPE = ",(0,n.jsx)(r.code,{children:"local"})]}),(0,n.jsx)(r.td,{children:"Path to the Expo public key"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"/path/to/public-key.pem"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/key-store#expo-signing-certificate",children:"Ref"})})]})]})]}),"\n",(0,n.jsxs)(r.h3,{id:"\ufe0f-aws--cloudfront-configuration",children:["\u2601\ufe0f ",(0,n.jsx)(r.strong,{children:"AWS & CloudFront Configuration"})]}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"AWS_REGION"})}),(0,n.jsxs)(r.td,{children:["\u2705 if using ",(0,n.jsx)(r.code,{children:"aws-secrets-manager"})," or ",(0,n.jsx)(r.code,{children:"s3"})]}),(0,n.jsx)(r.td,{children:"AWS Region"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"us-east-1"})}),(0,n.jsxs)(r.td,{children:[(0,n.jsx)(r.a,{href:"/docs/key-store?keyStore=aws-secrets-manager#key-store-configuration",children:"Ref"}),", ",(0,n.jsx)(r.a,{href:"/docs/storage?storage=s3",children:"Storage"})]})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"AWS_ACCESS_KEY_ID"})}),(0,n.jsxs)(r.td,{children:["\u2705 if using ",(0,n.jsx)(r.code,{children:"aws-secrets-manager"})," or ",(0,n.jsx)(r.code,{children:"s3"})," without IAM roles"]}),(0,n.jsx)(r.td,{children:"AWS Access Key ID"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"ACCESSKEYID"})}),(0,n.jsxs)(r.td,{children:[(0,n.jsx)(r.a,{href:"/docs/key-store?keyStore=aws-secrets-manager#key-store-configuration",children:"Ref"}),", ",(0,n.jsx)(r.a,{href:"/docs/storage?storage=s3",children:"Storage"})]})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"AWS_SECRET_ACCESS_KEY"})}),(0,n.jsxs)(r.td,{children:["\u2705 if using ",(0,n.jsx)(r.code,{children:"aws-secrets-manager"})," or ",(0,n.jsx)(r.code,{children:"s3"})," without IAM roles"]}),(0,n.jsx)(r.td,{children:"AWS Secret Access Key"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"SECRETACCESSKEY"})}),(0,n.jsxs)(r.td,{children:[(0,n.jsx)(r.a,{href:"/docs/key-store?keyStore=aws-secrets-manager#key-store-configuration",children:"Ref"}),", ",(0,n.jsx)(r.a,{href:"/docs/storage?storage=s3",children:"Storage"})]})]})]})]}),"\n",(0,n.jsx)(r.h4,{id:"cloudfront-settings",children:(0,n.jsx)(r.strong,{children:"CloudFront Settings"})}),"\n",(0,n.jsxs)(r.table,{children:[(0,n.jsx)(r.thead,{children:(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.th,{children:"Name"}),(0,n.jsx)(r.th,{children:"Required"}),(0,n.jsx)(r.th,{children:"Description"}),(0,n.jsx)(r.th,{children:"Example"}),(0,n.jsx)(r.th,{children:"Reference"})]})}),(0,n.jsxs)(r.tbody,{children:[(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"CLOUDFRONT_DOMAIN"})}),(0,n.jsx)(r.td,{children:"\u274c"}),(0,n.jsx)(r.td,{children:"CloudFront domain"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"https://XXX.cloudfront.net"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cdn/cloudfront",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"CLOUDFRONT_KEY_PAIR_ID"})}),(0,n.jsx)(r.td,{children:"\u2705 if CLOUDFRONT_DOMAIN is set"}),(0,n.jsx)(r.td,{children:"CloudFront key pair ID"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"Random string"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cdn/cloudfront",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"CLOUDFRONT_PRIVATE_KEY_B64"})}),(0,n.jsxs)(r.td,{children:["\u2705 if using ",(0,n.jsx)(r.code,{children:"environment"})," & CLOUDFRONT_DOMAIN is set"]}),(0,n.jsx)(r.td,{children:"Base64 CloudFront private key"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"Base64 string"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cdn/cloudfront",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID"})}),(0,n.jsxs)(r.td,{children:["\u2705 if using ",(0,n.jsx)(r.code,{children:"aws-secrets-manager"})," & CLOUDFRONT_DOMAIN is set"]}),(0,n.jsx)(r.td,{children:"CloudFront private key in AWS Secrets Manager"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"my-cloudfront-private-key"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cdn/cloudfront",children:"Ref"})})]}),(0,n.jsxs)(r.tr,{children:[(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"PRIVATE_LOCAL_CLOUDFRONT_KEY_PATH"})}),(0,n.jsxs)(r.td,{children:["\u2705 if using ",(0,n.jsx)(r.code,{children:"local"})," & CLOUDFRONT_DOMAIN is set"]}),(0,n.jsx)(r.td,{children:"Path to CloudFront private key"}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.code,{children:"/path/to/cloudfront-private-key.pem"})}),(0,n.jsx)(r.td,{children:(0,n.jsx)(r.a,{href:"/docs/cdn/cloudfront",children:"Ref"})})]})]})]})]})}function x(e={}){const{wrapper:r}={...(0,i.R)(),...e.components};return r?(0,n.jsx)(r,{...e,children:(0,n.jsx)(o,{...e})}):o(e)}},8453:(e,r,s)=>{s.d(r,{R:()=>c,x:()=>t});var d=s(6540);const n={},i=d.createContext(n);function c(e){const r=d.useContext(i);return d.useMemo((function(){return"function"==typeof e?e(r):{...r,...e}}),[r,e])}function t(e){let r;return r=e.disableParentContext?"function"==typeof e.components?e.components(n):e.components||n:c(e.components),d.createElement(i.Provider,{value:r},e.children)}}}]);
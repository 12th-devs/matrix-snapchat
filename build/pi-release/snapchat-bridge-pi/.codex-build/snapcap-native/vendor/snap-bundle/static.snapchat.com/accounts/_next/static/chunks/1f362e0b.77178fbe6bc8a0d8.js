"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[1261],{7730:function(e,t,n){n.d(t,{Eo:function(){return aJ},_s:function(){return dC},yu:function(){return aZ}});var o,r,i,a,s,l,c,u,d,h,p,f,m,g,v=n(84759),b=n(2784),y=n(96279),k=n(54004),w=n(31092),x=n(84336),S=n(42208),C=n(17213),F=n(42546),_=n(8634),N=n(30858),$=n(11752),E=n(21414),O=n(52322),D=n(98537),T=n(60019),z=n(1842);n(9850);var M=n(79857),I=n(12436),P=n(34291),j=n(70900),R=n(37025),L=n(41271),A=n(6795),V=n(37438),q=n(44173),Z=n(36941),B=n(47376),Q=n(68331),H=n(14868),G=n(96510),U=n(80645),W=n(16060),K=n(70895),Y=n(28114),X=n(33360),J=n(73980),ee=n(81139),et=n(46965),en=n(11439),eo=n(20193),er=n(81353),ei=n(81366),ea=n(86069),es=n(50768),el=n(29283),ec=n(57370),eu=n(74600),ed=n(28316),eh=n(27202),ep=n(83549),ef=n(99641),em=n(36402);n(18149);var eg=n(22589),ev=n(92070),eb=n(34406),ey=Object.defineProperty,ek=(e,t,n)=>t in e?ey(e,t,{enumerable:!0,configurable:!0,writable:!0,value:n}):e[t]=n,ew=(e,t)=>ey(e,"name",{value:t,configurable:!0}),ex=(e,t,n)=>(ek(e,"symbol"!=typeof t?t+"":t,n),n),eS={Click:"Click",LocaleSelect:"LocaleSelect"};function eC(e){let t;return e?e instanceof Error?e:"string"==typeof e?Error(e):((t="object"==typeof e?e.message?String(e.message):JSON.stringify(e):String(e)).length>100&&(t=`${t.substring(0,100)}...`),Error(t)):Error()}ew(eC,"parseError");var eF=class{constructor(e){ex(this,"clock"),ex(this,"resultCache"),ex(this,"onDataListeners",new Map),ex(this,"onErrorListeners",new Map),ex(this,"unsettledPromises",new Map),ex(this,"defaultExpiryTimeMs"),ex(this,"errorHandler"),ex(this,"register",ew(e=>{let t=this.clock();if(this.resultCache.has(e.dataId)){let n=this.resultCache.get(e.dataId);if(n.expiryTime>=t)return n.error?{error:n.error,hasLoaded:!0,isLoading:!1}:{data:n.data,hasLoaded:!0,isLoading:!1};this.delete(e.dataId)}if(this.onDataListeners.has(e.dataId)||this.onDataListeners.set(e.dataId,new Set),this.onErrorListeners.has(e.dataId)||this.onErrorListeners.set(e.dataId,new Set),e.onData&&this.onDataListeners.get(e.dataId).add(e.onData),e.onError&&this.onErrorListeners.get(e.dataId).add(e.onError),this.unsettledPromises.has(e.dataId))return{hasLoaded:!1,isLoading:!0};let n=e.dataAsync();return this.unsettledPromises.set(e.dataId,n),n.then(t=>{this.resultCache.set(e.dataId,{data:t,expiryTime:this.clock()+(e.ttlMs??this.defaultExpiryTimeMs)});let n=this.onDataListeners.get(e.dataId);n.forEach(e=>{try{e(t)}catch(e){this.errorHandler(e)}}),n.clear(),this.unsettledPromises.delete(e.dataId)}).catch(t=>{this.resultCache.set(e.dataId,{error:t,expiryTime:this.clock()+(e.ttlMs??3e4)});let n=this.onErrorListeners.get(e.dataId);n.forEach(e=>{try{e(t)}catch(e){this.errorHandler(e)}}),n.clear(),this.unsettledPromises.delete(e.dataId)}),{isLoading:!0,hasLoaded:!1}},"register")),ex(this,"countUnsettledPromises",ew(()=>this.unsettledPromises.size,"countUnsettledPromises")),ex(this,"unawaitedPromiseKeys",ew(()=>[...this.unsettledPromises.keys()],"unawaitedPromiseKeys")),ex(this,"allUnawaitedPromises",ew(async()=>{await Promise.allSettled(this.unsettledPromises.values())},"allUnawaitedPromises")),ex(this,"getCache",ew(()=>(this.cleanCache(),this.resultCache),"getCache")),ex(this,"cleanCache",ew(()=>{let e=0;return this.resultCache.forEach((t,n)=>{t.expiryTime<this.clock()&&(this.delete(n),e++)}),e},"cleanCache")),ex(this,"delete",ew(e=>{this.resultCache.delete(e),this.onDataListeners.delete(e),this.onErrorListeners.delete(e)},"delete")),this.clock=e.clock??Date.now,this.resultCache=e.cache??new Map,this.defaultExpiryTimeMs=e.defaultExpiryTimeMs??3e4,this.errorHandler=e.errorHandler??console.error}};ew(eF,"AsyncDataController");var e_=new eF({clock:Date.now}),eN={ar:"العربية","bn-BD":"বাংলা(বাংলাদেশ)","bn-IN":"বাংলা (ভারত)","bg-BG":"Bulg\xe1rach","zh-Hans":"中文简体","zh-Hant":"中文繁體","hr-HR":"Cr\xf3tach","cs-CZ":"Seiceach","da-DK":"Dansk","nl-NL":"Nederlands (Nederland)","en-GB":"English (UK)","en-US":"English (US)","et-EE":"East\xf3nach","fil-PH":"Filipino (Philippines)","fi-FI":"Suomi","fr-FR":"Fran\xe7ais (France)","de-DE":"Deutsch (Deutschland)","el-GR":"Ελληνικά","gu-IN":"ગુજરાતી","hi-IN":"हिन्दी","hu-HU":"Ung\xe1rach","id-ID":"Bahasa Indonesia","ga-IE":"\xc9ireannach","it-IT":"Italiano","ja-JP":"日本語","kn-IN":"ಕನ್ನಡ (India)","ko-KR":"한국어","lv-LV":"Laitviach","lt-LT":"Liotu\xe1nach","ms-MY":"Bahasa Melayu","ml-IN":"മലയാളം","mt-MT":"M\xe1ltach","mr-IN":"मराठी","nb-NO":"Norsk (bokm\xe5l)","pl-PL":"Polski","pt-BR":"Portugu\xeas (Brasil)","pt-PT":"Portugu\xeas (Portugal)",pa:"ਪੰਜਾਬੀ","ro-RO":"Rom\xe2nă","ru-RU":"Русский","sk-SK":"Sl\xf3vacach","sl-SI":"Sl\xf3iv\xe9anach","es-AR":"Espa\xf1ol (Argentine)",es:"Espa\xf1ol","es-MX":"Espa\xf1ol (M\xe9xico)","es-ES":"Espa\xf1ol (Espa\xf1a)","sv-SE":"Svenska","ta-IN":"தமிழ்","te-IN":"తెలుగు","th-TH":"ภาษาไทย (ประเทศไทย)","tr-TR":"T\xfcrk\xe7e","ur-PK":"اردو","vi-VN":"Tiếng Việt"},e$=["com","net","org","int","mil","edu","gov","io","as","co","ly","ht","ar","co.uk","app"],eE=ew(e=>{let t=e$.find(t=>e.endsWith(t));if(!t)return e;let n=e.substring(0,e.length-t.length-1).lastIndexOf(".");return n>0?`${e.substring(n+1)}`:e},"getTopLevelDomain"),eO=ew(function(e,t){return(eO=Object.setPrototypeOf||({__proto__:[]})instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var n in t)Object.prototype.hasOwnProperty.call(t,n)&&(e[n]=t[n])})(e,t)},"extendStatics");function eD(e,t){if("function"!=typeof t&&null!==t)throw TypeError("Class extends value "+String(t)+" is not a constructor or null");function n(){this.constructor=e}eO(e,t),ew(n,"__"),e.prototype=null===t?Object.create(t):(n.prototype=t.prototype,new n)}ew(eD,"__extends");var eT=ew(function(){return(eT=Object.assign||ew(function(e){for(var t,n=1,o=arguments.length;n<o;n++)for(var r in t=arguments[n])Object.prototype.hasOwnProperty.call(t,r)&&(e[r]=t[r]);return e},"__assign")).apply(this,arguments)},"__assign");function ez(e,t){var n={};for(var o in e)Object.prototype.hasOwnProperty.call(e,o)&&0>t.indexOf(o)&&(n[o]=e[o]);if(null!=e&&"function"==typeof Object.getOwnPropertySymbols)for(var r=0,o=Object.getOwnPropertySymbols(e);r<o.length;r++)0>t.indexOf(o[r])&&Object.prototype.propertyIsEnumerable.call(e,o[r])&&(n[o[r]]=e[o[r]]);return n}function eM(e,t,n,o){function r(e){return e instanceof n?e:new n(function(t){t(e)})}return ew(r,"adopt"),new(n||(n=Promise))(function(n,i){function a(e){try{l(o.next(e))}catch(e){i(e)}}function s(e){try{l(o.throw(e))}catch(e){i(e)}}function l(e){e.done?n(e.value):r(e.value).then(a,s)}ew(a,"fulfilled"),ew(s,"rejected"),ew(l,"step"),l((o=o.apply(e,t||[])).next())})}function eI(e,t){var n,o,r,i,a={label:0,sent:function(){if(1&r[0])throw r[1];return r[1]},trys:[],ops:[]};return i={next:s(0),throw:s(1),return:s(2)},"function"==typeof Symbol&&(i[Symbol.iterator]=function(){return this}),i;function s(i){return function(s){return function(i){if(n)throw TypeError("Generator is already executing.");for(;a;)try{if(n=1,o&&(r=2&i[0]?o.return:i[0]?o.throw||((r=o.return)&&r.call(o),0):o.next)&&!(r=r.call(o,i[1])).done)return r;switch(o=0,r&&(i=[2&i[0],r.value]),i[0]){case 0:case 1:r=i;break;case 4:return a.label++,{value:i[1],done:!1};case 5:a.label++,o=i[1],i=[0];continue;case 7:i=a.ops.pop(),a.trys.pop();continue;default:if(!(r=(r=a.trys).length>0&&r[r.length-1])&&(6===i[0]||2===i[0])){a=0;continue}if(3===i[0]&&(!r||i[1]>r[0]&&i[1]<r[3])){a.label=i[1];break}if(6===i[0]&&a.label<r[1]){a.label=r[1],r=i;break}if(r&&a.label<r[2]){a.label=r[2],a.ops.push(i);break}r[2]&&a.ops.pop(),a.trys.pop();continue}i=t.call(e,a)}catch(e){i=[6,e],o=0}finally{n=r=0}if(5&i[0])throw i[1];return{value:i[0]?i[1]:void 0,done:!0}}([i,s])}}}function eP(e,t,n){if(n||2==arguments.length)for(var o,r=0,i=t.length;r<i;r++)!o&&r in t||(o||(o=Array.prototype.slice.call(t,0,r)),o[r]=t[r]);return e.concat(o||Array.prototype.slice.call(t))}ew(ez,"__rest"),ew(eM,"__awaiter"),ew(eI,"__generator"),ew(eP,"__spreadArray");var ej="Invariant Violation",eR=Object.setPrototypeOf,eL=void 0===eR?function(e,t){return e.__proto__=t,e}:eR,eA=function(e){function t(n){void 0===n&&(n=ej);var o=e.call(this,"number"==typeof n?ej+": "+n+" (see https://github.com/apollographql/invariant-packages)":n)||this;return o.framesToPop=1,o.name=ej,eL(o,t.prototype),o}return eD(t,e),ew(t,"InvariantError"),t}(Error);function eV(e,t){if(!e)throw new eA(t)}ew(eV,"invariant");var eq=["debug","log","warn","error","silent"],eZ=eq.indexOf("log");function eB(e){return function(){if(eq.indexOf(e)>=eZ)return(console[e]||console.log).apply(console,arguments)}}function eQ(e){var t=eq[eZ];return eZ=Math.max(0,eq.indexOf(e)),t}function eH(e){try{return e()}catch(e){}}ew(eB,"wrapConsoleMethod"),(o=eV||(eV={})).debug=eB("debug"),o.log=eB("log"),o.warn=eB("warn"),o.error=eB("error"),ew(eQ,"setVerbosity"),ew(eH,"maybe");var eG=eH(function(){return globalThis})||eH(function(){return window})||eH(function(){return self})||eH(function(){return global})||eH(function(){return eH.constructor("return this")()}),eU="__DEV__";function eW(){try{return!!__DEV__}catch(e){return Object.defineProperty(eG,eU,{value:"production"!==eH(function(){return"production"}),enumerable:!1,configurable:!0,writable:!0}),eG[eU]}}ew(eW,"getDEV");var eK=eW();function eY(e){try{return e()}catch(e){}}ew(eY,"maybe");var eX=eY(function(){return globalThis})||eY(function(){return window})||eY(function(){return self})||eY(function(){return global})||eY(function(){return eY.constructor("return this")()}),eJ=!1;function e0(){!eX||eY(function(){return"production"})||eY(function(){return eb})||(Object.defineProperty(eX,"process",{value:{env:{NODE_ENV:"production"}},configurable:!0,enumerable:!1,writable:!0}),eJ=!0)}function e1(){eJ&&(delete eX.process,eJ=!1)}ew(e0,"install"),e0(),ew(e1,"remove");function e2(e,t){if(!e)throw Error(null!=t?t:"Unexpected invariant triggered.")}ew(e2,"invariant");var e3="function"==typeof Symbol&&"function"==typeof Symbol.for?Symbol.for("nodejs.util.inspect.custom"):void 0;function e5(e){var t=e.prototype.toJSON;"function"==typeof t||e2(0),e.prototype.inspect=t,e3&&(e.prototype[e3]=t)}function e4(e){return null!=e&&"string"==typeof e.kind}function e6(e){return(e6="function"==typeof Symbol&&"symbol"==typeof Symbol.iterator?ew(function(e){return typeof e},"_typeof"):ew(function(e){return e&&"function"==typeof Symbol&&e.constructor===Symbol&&e!==Symbol.prototype?"symbol":typeof e},"_typeof"))(e)}function e8(e){return e9(e,[])}function e9(e,t){switch(e6(e)){case"string":return JSON.stringify(e);case"function":return e.name?"[function ".concat(e.name,"]"):"[function]";case"object":if(null===e)return"null";return e7(e,t);default:return String(e)}}function e7(e,t){if(-1!==t.indexOf(e))return"[Circular]";var n=[].concat(t,[e]),o=tn(e);if(void 0!==o){var r=o.call(e);if(r!==e)return"string"==typeof r?r:e9(r,n)}else if(Array.isArray(e))return tt(e,n);return te(e,n)}function te(e,t){var n=Object.keys(e);return 0===n.length?"{}":t.length>2?"["+to(e)+"]":"{ "+n.map(function(n){var o=e9(e[n],t);return n+": "+o}).join(", ")+" }"}function tt(e,t){if(0===e.length)return"[]";if(t.length>2)return"[Array]";for(var n=Math.min(10,e.length),o=e.length-n,r=[],i=0;i<n;++i)r.push(e9(e[i],t));return 1===o?r.push("... 1 more item"):o>1&&r.push("... ".concat(o," more items")),"["+r.join(", ")+"]"}function tn(e){var t=e[String(e3)];return"function"==typeof t?t:"function"==typeof e.inspect?e.inspect:void 0}function to(e){var t=Object.prototype.toString.call(e).replace(/^\[object /,"").replace(/]$/,"");if("Object"===t&&"function"==typeof e.constructor){var n=e.constructor.name;if("string"==typeof n&&""!==n)return n}return t}function tr(e,t){for(var n=0;n<t.length;n++){var o=t[n];o.enumerable=o.enumerable||!1,o.configurable=!0,"value"in o&&(o.writable=!0),Object.defineProperty(e,o.key,o)}}function ti(e){var t=arguments.length>1&&void 0!==arguments[1]?arguments[1]:"",n=arguments.length>2&&void 0!==arguments[2]&&arguments[2],o=-1===e.indexOf("\n"),r=" "===e[0]||"	"===e[0],i='"'===e[e.length-1],a="\\"===e[e.length-1],s=!o||i||a||n,l="";return s&&!(o&&r)&&(l+="\n"+t),l+=t?e.replace(/\n/g,"\n"+t):e,s&&(l+="\n"),'"""'+l.replace(/"""/g,'\\"""')+'"""'}ew(e5,"defineInspect"),e5(function(){function e(e,t,n){this.start=e.start,this.end=t.end,this.startToken=e,this.endToken=t,this.source=n}return ew(e,"Location"),e.prototype.toJSON=ew(function(){return{start:this.start,end:this.end}},"toJSON"),e}()),e5(function(){function e(e,t,n,o,r,i,a){this.kind=e,this.start=t,this.end=n,this.line=o,this.column=r,this.value=a,this.prev=i,this.next=null}return ew(e,"Token"),e.prototype.toJSON=ew(function(){return{kind:this.kind,value:this.value,line:this.line,column:this.column}},"toJSON"),e}()),ew(e4,"isNode"),ew(e6,"_typeof"),ew(e8,"inspect"),ew(e9,"formatValue"),ew(e7,"formatObjectValue"),ew(te,"formatObject"),ew(tt,"formatArray"),ew(tn,"getCustomFn"),ew(to,"getObjectTag"),ew(function(e,t){if(!e)throw Error(t)},"devAssert"),ew(tr,"_defineProperties"),ew(function(e,t,n){return t&&tr(e.prototype,t),n&&tr(e,n),e},"_createClass"),ew(ti,"printBlockString");var ta={Name:[],Document:["definitions"],OperationDefinition:["name","variableDefinitions","directives","selectionSet"],VariableDefinition:["variable","type","defaultValue","directives"],Variable:["name"],SelectionSet:["selections"],Field:["alias","name","arguments","directives","selectionSet"],Argument:["name","value"],FragmentSpread:["name","directives"],InlineFragment:["typeCondition","directives","selectionSet"],FragmentDefinition:["name","variableDefinitions","typeCondition","directives","selectionSet"],IntValue:[],FloatValue:[],StringValue:[],BooleanValue:[],NullValue:[],EnumValue:[],ListValue:["values"],ObjectValue:["fields"],ObjectField:["name","value"],Directive:["name","arguments"],NamedType:["name"],ListType:["type"],NonNullType:["type"],SchemaDefinition:["description","directives","operationTypes"],OperationTypeDefinition:["type"],ScalarTypeDefinition:["description","name","directives"],ObjectTypeDefinition:["description","name","interfaces","directives","fields"],FieldDefinition:["description","name","arguments","type","directives"],InputValueDefinition:["description","name","type","defaultValue","directives"],InterfaceTypeDefinition:["description","name","interfaces","directives","fields"],UnionTypeDefinition:["description","name","directives","types"],EnumTypeDefinition:["description","name","directives","values"],EnumValueDefinition:["description","name","directives"],InputObjectTypeDefinition:["description","name","directives","fields"],DirectiveDefinition:["description","name","arguments","locations"],SchemaExtension:["directives","operationTypes"],ScalarTypeExtension:["name","directives"],ObjectTypeExtension:["name","interfaces","directives","fields"],InterfaceTypeExtension:["name","interfaces","directives","fields"],UnionTypeExtension:["name","directives","types"],EnumTypeExtension:["name","directives","values"],InputObjectTypeExtension:["name","directives","fields"]},ts=Object.freeze({});function tl(e,t){var n=arguments.length>2&&void 0!==arguments[2]?arguments[2]:ta,o=void 0,r=Array.isArray(e),i=[e],a=-1,s=[],l=void 0,c=void 0,u=void 0,d=[],h=[],p=e;do{var f,m=++a===i.length,g=m&&0!==s.length;if(m){if(c=0===h.length?void 0:d[d.length-1],l=u,u=h.pop(),g){if(r)l=l.slice();else{for(var v={},b=0,y=Object.keys(l);b<y.length;b++){var k=y[b];v[k]=l[k]}l=v}for(var w=0,x=0;x<s.length;x++){var S=s[x][0],C=s[x][1];r&&(S-=w),r&&null===C?(l.splice(S,1),w++):l[S]=C}}a=o.index,i=o.keys,s=o.edits,r=o.inArray,o=o.prev}else{if(c=u?r?a:i[a]:void 0,null==(l=u?u[c]:p))continue;u&&d.push(c)}var F=void 0;if(!Array.isArray(l)){if(!e4(l))throw Error("Invalid AST Node: ".concat(e8(l),"."));var _=tc(t,l.kind,m);if(_){if((F=_.call(t,l,c,u,d,h))===ts)break;if(!1===F){if(!m){d.pop();continue}}else if(void 0!==F&&(s.push([c,F]),!m)){if(e4(F))l=F;else{d.pop();continue}}}}void 0===F&&g&&s.push([c,l]),m?d.pop():(o={inArray:r,index:a,keys:i,edits:s,prev:o},i=(r=Array.isArray(l))?l:null!==(f=n[l.kind])&&void 0!==f?f:[],a=-1,s=[],u&&h.push(u),u=l)}while(void 0!==o);return 0!==s.length&&(p=s[s.length-1][1]),p}function tc(e,t,n){var o=e[t];if(o){if(!n&&"function"==typeof o)return o;var r=n?o.leave:o.enter;if("function"==typeof r)return r}else{var i=n?e.leave:e.enter;if(i){if("function"==typeof i)return i;var a=i[t];if("function"==typeof a)return a}}}function tu(e){return tl(e,{leave:td})}ew(tl,"visit"),ew(tc,"getVisitFn"),ew(tu,"print");var td={Name:ew(function(e){return e.value},"Name"),Variable:ew(function(e){return"$"+e.name},"Variable"),Document:ew(function(e){return tp(e.definitions,"\n\n")+"\n"},"Document"),OperationDefinition:ew(function(e){var t=e.operation,n=e.name,o=tm("(",tp(e.variableDefinitions,", "),")"),r=tp(e.directives," "),i=e.selectionSet;return n||r||o||"query"!==t?tp([t,tp([n,o]),r,i]," "):i},"OperationDefinition"),VariableDefinition:ew(function(e){var t=e.variable,n=e.type,o=e.defaultValue,r=e.directives;return t+": "+n+tm(" = ",o)+tm(" ",tp(r," "))},"VariableDefinition"),SelectionSet:ew(function(e){return tf(e.selections)},"SelectionSet"),Field:ew(function(e){var t=e.alias,n=e.name,o=e.arguments,r=e.directives,i=e.selectionSet,a=tm("",t,": ")+n,s=a+tm("(",tp(o,", "),")");return s.length>80&&(s=a+tm("(\n",tg(tp(o,"\n")),"\n)")),tp([s,tp(r," "),i]," ")},"Field"),Argument:ew(function(e){return e.name+": "+e.value},"Argument"),FragmentSpread:ew(function(e){return"..."+e.name+tm(" ",tp(e.directives," "))},"FragmentSpread"),InlineFragment:ew(function(e){var t=e.typeCondition,n=e.directives,o=e.selectionSet;return tp(["...",tm("on ",t),tp(n," "),o]," ")},"InlineFragment"),FragmentDefinition:ew(function(e){var t=e.name,n=e.typeCondition,o=e.variableDefinitions,r=e.directives,i=e.selectionSet;return"fragment ".concat(t).concat(tm("(",tp(o,", "),")")," ")+"on ".concat(n," ").concat(tm("",tp(r," ")," "))+i},"FragmentDefinition"),IntValue:ew(function(e){return e.value},"IntValue"),FloatValue:ew(function(e){return e.value},"FloatValue"),StringValue:ew(function(e,t){var n=e.value;return e.block?ti(n,"description"===t?"":"  "):JSON.stringify(n)},"StringValue"),BooleanValue:ew(function(e){return e.value?"true":"false"},"BooleanValue"),NullValue:ew(function(){return"null"},"NullValue"),EnumValue:ew(function(e){return e.value},"EnumValue"),ListValue:ew(function(e){return"["+tp(e.values,", ")+"]"},"ListValue"),ObjectValue:ew(function(e){return"{"+tp(e.fields,", ")+"}"},"ObjectValue"),ObjectField:ew(function(e){return e.name+": "+e.value},"ObjectField"),Directive:ew(function(e){return"@"+e.name+tm("(",tp(e.arguments,", "),")")},"Directive"),NamedType:ew(function(e){return e.name},"NamedType"),ListType:ew(function(e){return"["+e.type+"]"},"ListType"),NonNullType:ew(function(e){return e.type+"!"},"NonNullType"),SchemaDefinition:th(function(e){var t=e.directives,n=e.operationTypes;return tp(["schema",tp(t," "),tf(n)]," ")}),OperationTypeDefinition:ew(function(e){return e.operation+": "+e.type},"OperationTypeDefinition"),ScalarTypeDefinition:th(function(e){return tp(["scalar",e.name,tp(e.directives," ")]," ")}),ObjectTypeDefinition:th(function(e){var t=e.name,n=e.interfaces,o=e.directives,r=e.fields;return tp(["type",t,tm("implements ",tp(n," & ")),tp(o," "),tf(r)]," ")}),FieldDefinition:th(function(e){var t=e.name,n=e.arguments,o=e.type,r=e.directives;return t+(tb(n)?tm("(\n",tg(tp(n,"\n")),"\n)"):tm("(",tp(n,", "),")"))+": "+o+tm(" ",tp(r," "))}),InputValueDefinition:th(function(e){var t=e.name,n=e.type,o=e.defaultValue,r=e.directives;return tp([t+": "+n,tm("= ",o),tp(r," ")]," ")}),InterfaceTypeDefinition:th(function(e){var t=e.name,n=e.interfaces,o=e.directives,r=e.fields;return tp(["interface",t,tm("implements ",tp(n," & ")),tp(o," "),tf(r)]," ")}),UnionTypeDefinition:th(function(e){var t=e.name,n=e.directives,o=e.types;return tp(["union",t,tp(n," "),o&&0!==o.length?"= "+tp(o," | "):""]," ")}),EnumTypeDefinition:th(function(e){var t=e.name,n=e.directives,o=e.values;return tp(["enum",t,tp(n," "),tf(o)]," ")}),EnumValueDefinition:th(function(e){return tp([e.name,tp(e.directives," ")]," ")}),InputObjectTypeDefinition:th(function(e){var t=e.name,n=e.directives,o=e.fields;return tp(["input",t,tp(n," "),tf(o)]," ")}),DirectiveDefinition:th(function(e){var t=e.name,n=e.arguments,o=e.repeatable,r=e.locations;return"directive @"+t+(tb(n)?tm("(\n",tg(tp(n,"\n")),"\n)"):tm("(",tp(n,", "),")"))+(o?" repeatable":"")+" on "+tp(r," | ")}),SchemaExtension:ew(function(e){var t=e.directives,n=e.operationTypes;return tp(["extend schema",tp(t," "),tf(n)]," ")},"SchemaExtension"),ScalarTypeExtension:ew(function(e){return tp(["extend scalar",e.name,tp(e.directives," ")]," ")},"ScalarTypeExtension"),ObjectTypeExtension:ew(function(e){var t=e.name,n=e.interfaces,o=e.directives,r=e.fields;return tp(["extend type",t,tm("implements ",tp(n," & ")),tp(o," "),tf(r)]," ")},"ObjectTypeExtension"),InterfaceTypeExtension:ew(function(e){var t=e.name,n=e.interfaces,o=e.directives,r=e.fields;return tp(["extend interface",t,tm("implements ",tp(n," & ")),tp(o," "),tf(r)]," ")},"InterfaceTypeExtension"),UnionTypeExtension:ew(function(e){var t=e.name,n=e.directives,o=e.types;return tp(["extend union",t,tp(n," "),o&&0!==o.length?"= "+tp(o," | "):""]," ")},"UnionTypeExtension"),EnumTypeExtension:ew(function(e){var t=e.name,n=e.directives,o=e.values;return tp(["extend enum",t,tp(n," "),tf(o)]," ")},"EnumTypeExtension"),InputObjectTypeExtension:ew(function(e){var t=e.name,n=e.directives,o=e.fields;return tp(["extend input",t,tp(n," "),tf(o)]," ")},"InputObjectTypeExtension")};function th(e){return function(t){return tp([t.description,e(t)],"\n")}}function tp(e){var t,n=arguments.length>1&&void 0!==arguments[1]?arguments[1]:"";return null!==(t=null==e?void 0:e.filter(function(e){return e}).join(n))&&void 0!==t?t:""}function tf(e){return tm("{\n",tg(tp(e,"\n")),"\n}")}function tm(e,t){var n=arguments.length>2&&void 0!==arguments[2]?arguments[2]:"";return null!=t&&""!==t?e+t+n:""}function tg(e){return tm("  ",e.replace(/\n/g,"\n  "))}function tv(e){return -1!==e.indexOf("\n")}function tb(e){return null!=e&&e.some(tv)}function ty(){return e1()}function tk(){__DEV__?eV("boolean"==typeof eK,eK):eV("boolean"==typeof eK,36)}function tw(e,t){var n=e.directives;return!n||!n.length||t_(n).every(function(e){var n=e.directive,o=e.ifArgument,r=!1;return"Variable"===o.value.kind?(r=t&&t[o.value.name.value],__DEV__?eV(void 0!==r,"Invalid variable referenced in @".concat(n.name.value," directive.")):eV(void 0!==r,37)):r=o.value.value,"skip"===n.name.value?!r:r})}function tx(e){var t=[];return tl(e,{Directive:function(e){t.push(e.name.value)}}),t}function tS(e,t){return tx(t).some(function(t){return e.indexOf(t)>-1})}function tC(e){return e&&tS(["client"],e)&&tS(["export"],e)}function tF(e){var t=e.name.value;return"skip"===t||"include"===t}function t_(e){var t=[];return e&&e.length&&e.forEach(function(e){if(tF(e)){var n=e.arguments,o=e.name.value;__DEV__?eV(n&&1===n.length,"Incorrect number of arguments for the @".concat(o," directive.")):eV(n&&1===n.length,38);var r=n[0];__DEV__?eV(r.name&&"if"===r.name.value,"Invalid argument for the @".concat(o," directive.")):eV(r.name&&"if"===r.name.value,39);var i=r.value;__DEV__?eV(i&&("Variable"===i.kind||"BooleanValue"===i.kind),"Argument for the @".concat(o," directive must be a variable or a boolean value.")):eV(i&&("Variable"===i.kind||"BooleanValue"===i.kind),40),t.push({directive:e,ifArgument:r})}}),t}function tN(e,t){var n=t,o=[];return e.definitions.forEach(function(e){if("OperationDefinition"===e.kind)throw __DEV__?new eA("Found a ".concat(e.operation," operation").concat(e.name?" named '".concat(e.name.value,"'"):"",". ")+"No operations are allowed when using a fragment as a query. Only fragments are allowed."):new eA(41);"FragmentDefinition"===e.kind&&o.push(e)}),void 0===n&&(__DEV__?eV(1===o.length,"Found ".concat(o.length," fragments. `fragmentName` must be provided when there is not exactly 1 fragment.")):eV(1===o.length,42),n=o[0].name.value),eT(eT({},e),{definitions:eP([{kind:"OperationDefinition",operation:"query",selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:n}}]}}],e.definitions,!0)})}function t$(e){void 0===e&&(e=[]);var t={};return e.forEach(function(e){t[e.name.value]=e}),t}function tE(e,t){switch(e.kind){case"InlineFragment":return e;case"FragmentSpread":var n=t&&t[e.name.value];return __DEV__?eV(n,"No fragment named ".concat(e.name.value,".")):eV(n,43),n;default:return null}}function tO(e){return null!==e&&"object"==typeof e}function tD(e){return{__ref:String(e)}}function tT(e){return!!(e&&"object"==typeof e&&"string"==typeof e.__ref)}function tz(e){return tO(e)&&"Document"===e.kind&&Array.isArray(e.definitions)}function tM(e){return"StringValue"===e.kind}function tI(e){return"BooleanValue"===e.kind}function tP(e){return"IntValue"===e.kind}function tj(e){return"FloatValue"===e.kind}function tR(e){return"Variable"===e.kind}function tL(e){return"ObjectValue"===e.kind}function tA(e){return"ListValue"===e.kind}function tV(e){return"EnumValue"===e.kind}function tq(e){return"NullValue"===e.kind}function tZ(e,t,n,o){if(tP(n)||tj(n))e[t.value]=Number(n.value);else if(tI(n)||tM(n))e[t.value]=n.value;else if(tL(n)){var r={};n.fields.map(function(e){return tZ(r,e.name,e.value,o)}),e[t.value]=r}else if(tR(n)){var i=(o||{})[n.name.value];e[t.value]=i}else if(tA(n))e[t.value]=n.values.map(function(e){var n={};return tZ(n,t,e,o),n[t.value]});else if(tV(n))e[t.value]=n.value;else if(tq(n))e[t.value]=null;else throw __DEV__?new eA('The inline argument "'.concat(t.value,'" of kind "').concat(n.kind,'"')+"is not supported. Use variables instead of inline arguments to overcome this limitation."):new eA(52)}function tB(e,t){var n=null;e.directives&&(n={},e.directives.forEach(function(e){n[e.name.value]={},e.arguments&&e.arguments.forEach(function(o){var r=o.name,i=o.value;return tZ(n[e.name.value],r,i,t)})}));var o=null;return e.arguments&&e.arguments.length&&(o={},e.arguments.forEach(function(e){var n=e.name,r=e.value;return tZ(o,n,r,t)})),tH(e.name.value,o,n)}ew(th,"addDescription"),ew(tp,"join"),ew(tf,"block"),ew(tm,"wrap"),ew(tg,"indent"),ew(tv,"isMultiline"),ew(tb,"hasMultilineItems"),ew(ty,"removeTemporaryGlobals"),ew(tk,"checkDEV"),ty(),tk(),ew(tw,"shouldInclude"),ew(tx,"getDirectiveNames"),ew(tS,"hasDirectives"),ew(tC,"hasClientExports"),ew(tF,"isInclusionDirective"),ew(t_,"getInclusionDirectives"),ew(tN,"getFragmentQueryDocument"),ew(t$,"createFragmentMap"),ew(tE,"getFragmentFromSelection"),ew(tO,"isNonNullObject"),ew(tD,"makeReference"),ew(tT,"isReference"),ew(tz,"isDocumentNode"),ew(tM,"isStringValue"),ew(tI,"isBooleanValue"),ew(tP,"isIntValue"),ew(tj,"isFloatValue"),ew(tR,"isVariable"),ew(tL,"isObjectValue"),ew(tA,"isListValue"),ew(tV,"isEnumValue"),ew(tq,"isNullValue"),ew(tZ,"valueToObjectRepresentation"),ew(tB,"storeKeyNameFromField");var tQ=["connection","include","skip","client","rest","export"],tH=Object.assign(function(e,t,n){if(t&&n&&n.connection&&n.connection.key){if(!n.connection.filter||!(n.connection.filter.length>0))return n.connection.key;var o=n.connection.filter?n.connection.filter:[];o.sort();var r={};return o.forEach(function(e){r[e]=t[e]}),"".concat(n.connection.key,"(").concat(tG(r),")")}var i=e;if(t){var a=tG(t);i+="(".concat(a,")")}return n&&Object.keys(n).forEach(function(e){-1===tQ.indexOf(e)&&(n[e]&&Object.keys(n[e]).length?i+="@".concat(e,"(").concat(tG(n[e]),")"):i+="@".concat(e))}),i},{setStringify:function(e){var t=tG;return tG=e,t}}),tG=ew(function(e){return JSON.stringify(e,tU)},"defaultStringify");function tU(e,t){return tO(t)&&!Array.isArray(t)&&(t=Object.keys(t).sort().reduce(function(e,n){return e[n]=t[n],e},{})),t}function tW(e,t){if(e.arguments&&e.arguments.length){var n={};return e.arguments.forEach(function(e){return tZ(n,e.name,e.value,t)}),n}return null}function tK(e){return e.alias?e.alias.value:e.name.value}function tY(e,t,n){if("string"==typeof e.__typename)return e.__typename;for(var o=0,r=t.selections;o<r.length;o++){var i=r[o];if(tX(i)){if("__typename"===i.name.value)return e[tK(i)]}else{var a=tY(e,tE(i,n).selectionSet,n);if("string"==typeof a)return a}}}function tX(e){return"Field"===e.kind}function tJ(e){return"InlineFragment"===e.kind}function t0(e){__DEV__?eV(e&&"Document"===e.kind,'Expecting a parsed GraphQL document. Perhaps you need to wrap the query string in a "gql" tag? http://docs.apollostack.com/apollo-client/core.html#gql'):eV(e&&"Document"===e.kind,44);var t=e.definitions.filter(function(e){return"FragmentDefinition"!==e.kind}).map(function(e){if("OperationDefinition"!==e.kind)throw __DEV__?new eA('Schema type definitions not allowed in queries. Found: "'.concat(e.kind,'"')):new eA(45);return e});return __DEV__?eV(t.length<=1,"Ambiguous GraphQL document: contains ".concat(t.length," operations")):eV(t.length<=1,46),e}function t1(e){return t0(e),e.definitions.filter(function(e){return"OperationDefinition"===e.kind})[0]}function t2(e){return e.definitions.filter(function(e){return"OperationDefinition"===e.kind&&e.name}).map(function(e){return e.name.value})[0]||null}function t3(e){return e.definitions.filter(function(e){return"FragmentDefinition"===e.kind})}function t5(e){var t=t1(e);return __DEV__?eV(t&&"query"===t.operation,"Must contain a query definition."):eV(t&&"query"===t.operation,47),t}function t4(e){__DEV__?eV("Document"===e.kind,'Expecting a parsed GraphQL document. Perhaps you need to wrap the query string in a "gql" tag? http://docs.apollostack.com/apollo-client/core.html#gql'):eV("Document"===e.kind,48),__DEV__?eV(e.definitions.length<=1,"Fragment must have exactly one definition."):eV(e.definitions.length<=1,49);var t=e.definitions[0];return __DEV__?eV("FragmentDefinition"===t.kind,"Must be a fragment definition."):eV("FragmentDefinition"===t.kind,50),t}function t6(e){t0(e);for(var t,n=0,o=e.definitions;n<o.length;n++){var r=o[n];if("OperationDefinition"===r.kind){var i=r.operation;if("query"===i||"mutation"===i||"subscription"===i)return r}"FragmentDefinition"!==r.kind||t||(t=r)}if(t)return t;throw __DEV__?new eA("Expected a parsed GraphQL query with a query, mutation, subscription, or a fragment."):new eA(51)}function t8(e){var t=Object.create(null),n=e&&e.variableDefinitions;return n&&n.length&&n.forEach(function(e){e.defaultValue&&tZ(t,e.variable.name,e.defaultValue)}),t}function t9(e,t,n){var o=0;return e.forEach(function(n,r){t.call(this,n,r,e)&&(e[o++]=n)},n),e.length=o,e}ew(tU,"stringifyReplacer"),ew(tW,"argumentsObjectFromField"),ew(tK,"resultKeyNameFromField"),ew(tY,"getTypenameFromResult"),ew(tX,"isField"),ew(tJ,"isInlineFragment"),ew(t0,"checkDocument"),ew(t1,"getOperationDefinition"),ew(t2,"getOperationName"),ew(t3,"getFragmentDefinitions"),ew(t5,"getQueryDefinition"),ew(t4,"getFragmentDefinition"),ew(t6,"getMainDefinition"),ew(t8,"getDefaultValues"),ew(t9,"filterInPlace");var t7={kind:"Field",name:{kind:"Name",value:"__typename"}};function ne(e,t){return e.selectionSet.selections.every(function(e){return"FragmentSpread"===e.kind&&ne(t[e.name.value],t)})}function nt(e){return ne(t1(e)||t4(e),t$(t3(e)))?null:e}function nn(e){return ew(function(t){return e.some(function(e){return e.name&&e.name===t.name.value||e.test&&e.test(t)})},"directiveMatcher")}function no(e,t){var n=Object.create(null),o=[],r=Object.create(null),i=[],a=nt(tl(t,{Variable:{enter:function(e,t,o){"VariableDefinition"!==o.kind&&(n[e.name.value]=!0)}},Field:{enter:function(t){if(e&&t.directives&&e.some(function(e){return e.remove})&&t.directives&&t.directives.some(nn(e)))return t.arguments&&t.arguments.forEach(function(e){"Variable"===e.value.kind&&o.push({name:e.value.name.value})}),t.selectionSet&&nu(t.selectionSet).forEach(function(e){i.push({name:e.name.value})}),null}},FragmentSpread:{enter:function(e){r[e.name.value]=!0}},Directive:{enter:function(t){if(nn(e)(t))return null}}}));return a&&t9(o,function(e){return!!e.name&&!n[e.name]}).length&&(a=nl(o,a)),a&&t9(i,function(e){return!!e.name&&!r[e.name]}).length&&(a=nc(i,a)),a}ew(ne,"isEmpty"),ew(nt,"nullIfDocIsEmpty"),ew(nn,"getDirectiveMatcher"),ew(no,"removeDirectivesFromDocument");var nr=Object.assign(function(e){return tl(e,{SelectionSet:{enter:function(e,t,n){if(!n||"OperationDefinition"!==n.kind){var o=e.selections;if(!(!o||o.some(function(e){return tX(e)&&("__typename"===e.name.value||0===e.name.value.lastIndexOf("__",0))}))&&!(tX(n)&&n.directives&&n.directives.some(function(e){return"export"===e.name.value})))return eT(eT({},e),{selections:eP(eP([],o,!0),[t7],!1)})}}}})},{added:function(e){return e===t7}}),ni={test:function(e){var t="connection"===e.name.value;return t&&(!e.arguments||!e.arguments.some(function(e){return"key"===e.name.value}))&&__DEV__&&eV.warn("Removing an @connection directive even though it does not have a key. You may want to use the key parameter to specify a store key."),t}};function na(e){return no([ni],t0(e))}function ns(e){return ew(function(t){return e.some(function(e){return t.value&&"Variable"===t.value.kind&&t.value.name&&(e.name===t.value.name.value||e.test&&e.test(t))})},"argumentMatcher")}function nl(e,t){var n=ns(e);return nt(tl(t,{OperationDefinition:{enter:function(t){return eT(eT({},t),{variableDefinitions:t.variableDefinitions?t.variableDefinitions.filter(function(t){return!e.some(function(e){return e.name===t.variable.name.value})}):[]})}},Field:{enter:function(t){if(e.some(function(e){return e.remove})){var o=0;if(t.arguments&&t.arguments.forEach(function(e){n(e)&&(o+=1)}),1===o)return null}}},Argument:{enter:function(e){if(n(e))return null}}}))}function nc(e,t){function n(t){if(e.some(function(e){return e.name===t.name.value}))return null}return ew(n,"enter"),nt(tl(t,{FragmentSpread:{enter:n},FragmentDefinition:{enter:n}}))}function nu(e){var t=[];return e.selections.forEach(function(e){(tX(e)||tJ(e))&&e.selectionSet?nu(e.selectionSet).forEach(function(e){return t.push(e)}):"FragmentSpread"===e.kind&&t.push(e)}),t}function nd(e){return"query"===t6(e).operation?e:tl(e,{OperationDefinition:{enter:function(e){return eT(eT({},e),{operation:"query"})}}})}function nh(e){t0(e);var t=no([{test:function(e){return"client"===e.name.value},remove:!0}],e);return t&&(t=tl(t,{FragmentDefinition:{enter:function(e){if(e.selectionSet&&e.selectionSet.selections.every(function(e){return tX(e)&&"__typename"===e.name.value}))return null}}})),t}ew(na,"removeConnectionDirectiveFromDocument"),ew(ns,"getArgumentMatcher"),ew(nl,"removeArgumentsFromDocument"),ew(nc,"removeFragmentSpreadFromDocument"),ew(nu,"getAllFragmentSpreadsFromSelectionSet"),ew(nd,"buildQueryFromSelectionSet"),ew(nh,"removeClientSetsFromDocument");var np=Object.prototype.hasOwnProperty;function nf(){for(var e=[],t=0;t<arguments.length;t++)e[t]=arguments[t];return nm(e)}function nm(e){var t=e[0]||{},n=e.length;if(n>1)for(var o=new nv,r=1;r<n;++r)t=o.merge(t,e[r]);return t}ew(nf,"mergeDeep"),ew(nm,"mergeDeepArray");var ng=ew(function(e,t,n){return this.merge(e[n],t[n])},"defaultReconciler"),nv=function(){function e(e){void 0===e&&(e=ng),this.reconciler=e,this.isObject=tO,this.pastCopies=new Set}return ew(e,"DeepMerger"),e.prototype.merge=function(e,t){for(var n=this,o=[],r=2;r<arguments.length;r++)o[r-2]=arguments[r];return tO(t)&&tO(e)?(Object.keys(t).forEach(function(r){if(np.call(e,r)){var i=e[r];if(t[r]!==i){var a=n.reconciler.apply(n,eP([e,t,r],o,!1));a!==i&&((e=n.shallowCopyForMerge(e))[r]=a)}}else(e=n.shallowCopyForMerge(e))[r]=t[r]}),e):t},e.prototype.shallowCopyForMerge=function(e){return tO(e)&&!this.pastCopies.has(e)&&(e=Array.isArray(e)?e.slice(0):eT({__proto__:Object.getPrototypeOf(e)},e),this.pastCopies.add(e)),e},e}();function nb(e,t){var n="undefined"!=typeof Symbol&&e[Symbol.iterator]||e["@@iterator"];if(n)return(n=n.call(e)).next.bind(n);if(Array.isArray(e)||(n=ny(e))||t&&e&&"number"==typeof e.length){n&&(e=n);var o=0;return function(){return o>=e.length?{done:!0}:{done:!1,value:e[o++]}}}throw TypeError("Invalid attempt to iterate non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method.")}function ny(e,t){if(e){if("string"==typeof e)return nk(e,t);var n=Object.prototype.toString.call(e).slice(8,-1);if("Object"===n&&e.constructor&&(n=e.constructor.name),"Map"===n||"Set"===n)return Array.from(e);if("Arguments"===n||/^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(n))return nk(e,t)}}function nk(e,t){(null==t||t>e.length)&&(t=e.length);for(var n=0,o=Array(t);n<t;n++)o[n]=e[n];return o}function nw(e,t){for(var n=0;n<t.length;n++){var o=t[n];o.enumerable=o.enumerable||!1,o.configurable=!0,"value"in o&&(o.writable=!0),Object.defineProperty(e,o.key,o)}}function nx(e,t,n){return t&&nw(e.prototype,t),n&&nw(e,n),Object.defineProperty(e,"prototype",{writable:!1}),e}ew(nb,"_createForOfIteratorHelperLoose"),ew(ny,"_unsupportedIterableToArray"),ew(nk,"_arrayLikeToArray"),ew(nw,"_defineProperties"),ew(nx,"_createClass");var nS=ew(function(){return"function"==typeof Symbol},"hasSymbols"),nC=ew(function(e){return nS()&&!!Symbol[e]},"hasSymbol"),nF=ew(function(e){return nC(e)?Symbol[e]:"@@"+e},"getSymbol");nS()&&!nC("observable")&&(Symbol.observable=Symbol("observable"));var n_=nF("iterator"),nN=nF("observable"),n$=nF("species");function nE(e,t){var n=e[t];if(null!=n){if("function"!=typeof n)throw TypeError(n+" is not a function");return n}}function nO(e){var t=e.constructor;return void 0!==t&&null===(t=t[n$])&&(t=void 0),void 0!==t?t:nV}function nD(e){return e instanceof nV}function nT(e){nT.log?nT.log(e):setTimeout(function(){throw e})}function nz(e){Promise.resolve().then(function(){try{e()}catch(e){nT(e)}})}function nM(e){var t=e._cleanup;if(void 0!==t){if(e._cleanup=void 0,!t)return;try{if("function"==typeof t)t();else{var n=nE(t,"unsubscribe");n&&n.call(t)}}catch(e){nT(e)}}}function nI(e){e._observer=void 0,e._queue=void 0,e._state="closed"}function nP(e){var t=e._queue;if(t){e._queue=void 0,e._state="ready";for(var n=0;n<t.length&&(nj(e,t[n].type,t[n].value),"closed"!==e._state);++n);}}function nj(e,t,n){e._state="running";var o=e._observer;try{var r=nE(o,t);switch(t){case"next":r&&r.call(o,n);break;case"error":if(nI(e),r)r.call(o,n);else throw n;break;case"complete":nI(e),r&&r.call(o)}}catch(e){nT(e)}"closed"===e._state?nM(e):"running"===e._state&&(e._state="ready")}function nR(e,t,n){if("closed"!==e._state){if("buffering"===e._state){e._queue.push({type:t,value:n});return}if("ready"!==e._state){e._state="buffering",e._queue=[{type:t,value:n}],nz(function(){return nP(e)});return}nj(e,t,n)}}ew(nE,"getMethod"),ew(nO,"getSpecies"),ew(nD,"isObservable"),ew(nT,"hostReportError"),ew(nz,"enqueue"),ew(nM,"cleanupSubscription"),ew(nI,"closeSubscription"),ew(nP,"flushSubscription"),ew(nj,"notifySubscription"),ew(nR,"onNotify");var nL=function(){function e(e,t){this._cleanup=void 0,this._observer=e,this._queue=void 0,this._state="initializing";var n=new nA(this);try{this._cleanup=t.call(void 0,n)}catch(e){n.error(e)}"initializing"===this._state&&(this._state="ready")}return ew(e,"Subscription"),e.prototype.unsubscribe=ew(function(){"closed"!==this._state&&(nI(this),nM(this))},"unsubscribe"),nx(e,[{key:"closed",get:function(){return"closed"===this._state}}]),e}(),nA=function(){function e(e){this._subscription=e}ew(e,"SubscriptionObserver");var t=e.prototype;return t.next=ew(function(e){nR(this._subscription,"next",e)},"next"),t.error=ew(function(e){nR(this._subscription,"error",e)},"error"),t.complete=ew(function(){nR(this._subscription,"complete")},"complete"),nx(e,[{key:"closed",get:function(){return"closed"===this._subscription._state}}]),e}(),nV=function(){function e(t){if(!(this instanceof e))throw TypeError("Observable cannot be called as a function");if("function"!=typeof t)throw TypeError("Observable initializer must be a function");this._subscriber=t}ew(e,"Observable");var t=e.prototype;return t.subscribe=ew(function(e){return("object"!=typeof e||null===e)&&(e={next:e,error:arguments[1],complete:arguments[2]}),new nL(e,this._subscriber)},"subscribe"),t.forEach=ew(function(e){var t=this;return new Promise(function(n,o){if("function"!=typeof e){o(TypeError(e+" is not a function"));return}function r(){i.unsubscribe(),n()}ew(r,"done");var i=t.subscribe({next:function(t){try{e(t,r)}catch(e){o(e),i.unsubscribe()}},error:o,complete:n})})},"forEach"),t.map=ew(function(e){var t=this;if("function"!=typeof e)throw TypeError(e+" is not a function");return new(nO(this))(function(n){return t.subscribe({next:function(t){try{t=e(t)}catch(e){return n.error(e)}n.next(t)},error:function(e){n.error(e)},complete:function(){n.complete()}})})},"map"),t.filter=ew(function(e){var t=this;if("function"!=typeof e)throw TypeError(e+" is not a function");return new(nO(this))(function(n){return t.subscribe({next:function(t){try{if(!e(t))return}catch(e){return n.error(e)}n.next(t)},error:function(e){n.error(e)},complete:function(){n.complete()}})})},"filter"),t.reduce=ew(function(e){var t=this;if("function"!=typeof e)throw TypeError(e+" is not a function");var n=nO(this),o=arguments.length>1,r=!1,i=arguments[1],a=i;return new n(function(n){return t.subscribe({next:function(t){var i=!r;if(r=!0,!i||o)try{a=e(a,t)}catch(e){return n.error(e)}else a=t},error:function(e){n.error(e)},complete:function(){if(!r&&!o)return n.error(TypeError("Cannot reduce an empty sequence"));n.next(a),n.complete()}})})},"reduce"),t.concat=ew(function(){for(var e=this,t=arguments.length,n=Array(t),o=0;o<t;o++)n[o]=arguments[o];var r=nO(this);return new r(function(t){var o,i=0;function a(e){o=e.subscribe({next:function(e){t.next(e)},error:function(e){t.error(e)},complete:function(){i===n.length?(o=void 0,t.complete()):a(r.from(n[i++]))}})}return ew(a,"startNext"),a(e),function(){o&&(o.unsubscribe(),o=void 0)}})},"concat"),t.flatMap=ew(function(e){var t=this;if("function"!=typeof e)throw TypeError(e+" is not a function");var n=nO(this);return new n(function(o){var r=[],i=t.subscribe({next:function(t){if(e)try{t=e(t)}catch(e){return o.error(e)}var i=n.from(t).subscribe({next:function(e){o.next(e)},error:function(e){o.error(e)},complete:function(){var e=r.indexOf(i);e>=0&&r.splice(e,1),a()}});r.push(i)},error:function(e){o.error(e)},complete:function(){a()}});function a(){i.closed&&0===r.length&&o.complete()}return ew(a,"completeIfDone"),function(){r.forEach(function(e){return e.unsubscribe()}),i.unsubscribe()}})},"flatMap"),t[nN]=function(){return this},e.from=ew(function(t){var n="function"==typeof this?this:e;if(null==t)throw TypeError(t+" is not an object");var o=nE(t,nN);if(o){var r=o.call(t);if(Object(r)!==r)throw TypeError(r+" is not an object");return nD(r)&&r.constructor===n?r:new n(function(e){return r.subscribe(e)})}if(nC("iterator")&&(o=nE(t,n_)))return new n(function(e){nz(function(){if(!e.closed){for(var n,r=nb(o.call(t));!(n=r()).done;){var i=n.value;if(e.next(i),e.closed)return}e.complete()}})});if(Array.isArray(t))return new n(function(e){nz(function(){if(!e.closed){for(var n=0;n<t.length;++n)if(e.next(t[n]),e.closed)return;e.complete()}})});throw TypeError(t+" is not observable")},"from"),e.of=ew(function(){for(var t=arguments.length,n=Array(t),o=0;o<t;o++)n[o]=arguments[o];return new("function"==typeof this?this:e)(function(e){nz(function(){if(!e.closed){for(var t=0;t<n.length;++t)if(e.next(n[t]),e.closed)return;e.complete()}})})},"of"),nx(e,null,[{key:n$,get:function(){return this}}]),e}();function nq(e){var t,n=e.Symbol;if("function"==typeof n){if(n.observable)t=n.observable;else{t="function"==typeof n.for?n.for("https://github.com/benlesh/symbol-observable"):n("https://github.com/benlesh/symbol-observable");try{n.observable=t}catch(e){}}}else t="@@observable";return t}nS()&&Object.defineProperty(nV,Symbol("extensions"),{value:{symbol:nN,hostReportError:nT},configurable:!0}),ew(nq,"symbolObservablePonyfill"),"undefined"!=typeof self?d=self:"undefined"!=typeof window?d=window:"undefined"!=typeof global?d=global:"undefined"!=typeof module?d=module:d=Function("return this")(),nq(d);var nZ=nV.prototype,nB="@@observable";nZ[nB]||(nZ[nB]=function(){return this});var nQ=Object.prototype.toString;function nH(e){return nG(e)}function nG(e,t){switch(nQ.call(e)){case"[object Array]":if((t=t||new Map).has(e))return t.get(e);var n=e.slice(0);return t.set(e,n),n.forEach(function(e,o){n[o]=nG(e,t)}),n;case"[object Object]":if((t=t||new Map).has(e))return t.get(e);var o=Object.create(Object.getPrototypeOf(e));return t.set(e,o),Object.keys(e).forEach(function(n){o[n]=nG(e[n],t)}),o;default:return e}}function nU(e){var t=new Set([e]);return t.forEach(function(e){tO(e)&&nW(e)===e&&Object.getOwnPropertyNames(e).forEach(function(n){tO(e[n])&&t.add(e[n])})}),e}function nW(e){if(__DEV__&&!Object.isFrozen(e))try{Object.freeze(e)}catch(e){if(e instanceof TypeError)return null;throw e}return e}function nK(e){return __DEV__&&nU(e),e}function nY(e,t,n){var o=[];e.forEach(function(e){return e[t]&&o.push(e)}),o.forEach(function(e){return e[t](n)})}function nX(e,t,n){return new nV(function(o){var r=o.next,i=o.error,a=o.complete,s=0,l=!1,c={then:function(e){return new Promise(function(t){return t(e())})}};function u(e,t){return e?function(t){++s;var n=ew(function(){return e(t)},"both");c=c.then(n,n).then(function(e){--s,r&&r.call(o,e),l&&d.complete()},function(e){throw--s,e}).catch(function(e){i&&i.call(o,e)})}:function(e){return t&&t.call(o,e)}}ew(u,"makeCallback");var d={next:u(t,r),error:u(n,i),complete:function(){l=!0,!s&&a&&a.call(o)}},h=e.subscribe(d);return function(){return h.unsubscribe()}})}ew(nH,"cloneDeep"),ew(nG,"cloneDeepHelper"),ew(nU,"deepFreeze"),ew(nW,"shallowFreeze"),ew(nK,"maybeDeepFreeze"),ew(nY,"iterateObserversSafely"),ew(nX,"asyncMap");var nJ="function"==typeof WeakMap&&"ReactNative"!==eH(function(){return navigator.product}),n0="function"==typeof WeakSet,n1="function"==typeof Symbol&&"function"==typeof Symbol.for,n2="function"==typeof eH(function(){return window.document.createElement}),n3=eH(function(){return navigator.userAgent.indexOf("jsdom")>=0})||!1,n5=n2&&!n3;function n4(e){function t(t){Object.defineProperty(e,t,{value:nV})}return ew(t,"set"),n1&&Symbol.species&&t(Symbol.species),t("@@species"),e}function n6(e){return e&&"function"==typeof e.then}ew(n4,"fixObservableSubclass"),ew(n6,"isPromiseLike");var n8=function(e){function t(t){var n=e.call(this,function(e){return n.addObserver(e),function(){return n.removeObserver(e)}})||this;return n.observers=new Set,n.addCount=0,n.promise=new Promise(function(e,t){n.resolve=e,n.reject=t}),n.handlers={next:function(e){null!==n.sub&&(n.latest=["next",e],nY(n.observers,"next",e))},error:function(e){var t=n.sub;null!==t&&(t&&setTimeout(function(){return t.unsubscribe()}),n.sub=null,n.latest=["error",e],n.reject(e),nY(n.observers,"error",e))},complete:function(){var e=n.sub;if(null!==e){var t=n.sources.shift();t?n6(t)?t.then(function(e){return n.sub=e.subscribe(n.handlers)}):n.sub=t.subscribe(n.handlers):(e&&setTimeout(function(){return e.unsubscribe()}),n.sub=null,n.latest&&"next"===n.latest[0]?n.resolve(n.latest[1]):n.resolve(),nY(n.observers,"complete"))}}},n.cancel=function(e){n.reject(e),n.sources=[],n.handlers.complete()},n.promise.catch(function(e){}),"function"==typeof t&&(t=[new nV(t)]),n6(t)?t.then(function(e){return n.start(e)},n.handlers.error):n.start(t),n}return eD(t,e),ew(t,"Concast"),t.prototype.start=function(e){void 0===this.sub&&(this.sources=Array.from(e),this.handlers.complete())},t.prototype.deliverLastMessage=function(e){if(this.latest){var t=this.latest[0],n=e[t];n&&n.call(e,this.latest[1]),null===this.sub&&"next"===t&&e.complete&&e.complete()}},t.prototype.addObserver=function(e){!this.observers.has(e)&&(this.deliverLastMessage(e),this.observers.add(e),++this.addCount)},t.prototype.removeObserver=function(e,t){this.observers.delete(e)&&--this.addCount<1&&!t&&this.handlers.complete()},t.prototype.cleanup=function(e){var t=this,n=!1,o=ew(function(){n||(n=!0,t.observers.delete(r),e())},"once"),r={next:o,error:o,complete:o},i=this.addCount;this.addObserver(r),this.addCount=i},t}(nV);function n9(e){return Array.isArray(e)&&e.length>0}function n7(e){return e.errors&&e.errors.length>0||!1}function oe(){for(var e=[],t=0;t<arguments.length;t++)e[t]=arguments[t];var n=Object.create(null);return e.forEach(function(e){e&&Object.keys(e).forEach(function(t){var o=e[t];void 0!==o&&(n[t]=o)})}),n}n4(n8),ew(n9,"isNonEmptyArray"),ew(n7,"graphQLResultHasError"),ew(oe,"compact");var ot=new Map;function on(e){var t=ot.get(e)||1;return ot.set(e,t+1),"".concat(e,":").concat(t,":").concat(Math.random().toString(36).slice(2))}function oo(e){var t=on("stringifyForDisplay");return JSON.stringify(e,function(e,n){return void 0===n?t:n}).split(JSON.stringify(t)).join("<undefined>")}function or(e,t){return oe(e,t,t.variables&&{variables:eT(eT({},e&&e.variables),t.variables)})}function oi(e){return new nV(function(t){t.error(e)})}ew(on,"makeUniqueId"),ew(oo,"stringifyForDisplay"),ew(or,"mergeOptions"),ew(oi,"fromError");var oa=ew(function(e,t,n){var o=Error(n);throw o.name="ServerError",o.response=e,o.statusCode=e.status,o.result=t,o},"throwServerError");function os(e){for(var t=["query","operationName","variables","extensions","context"],n=0,o=Object.keys(e);n<o.length;n++){var r=o[n];if(0>t.indexOf(r))throw __DEV__?new eA("illegal argument: ".concat(r)):new eA(24)}return e}function ol(e,t){var n=eT({},e),o=ew(function(e){n="function"==typeof e?eT(eT({},n),e(n)):eT(eT({},n),e)},"setContext"),r=ew(function(){return eT({},n)},"getContext");return Object.defineProperty(t,"setContext",{enumerable:!1,value:o}),Object.defineProperty(t,"getContext",{enumerable:!1,value:r}),t}function oc(e){var t={variables:e.variables||{},extensions:e.extensions||{},operationName:e.operationName,query:e.query};return t.operationName||(t.operationName="string"!=typeof t.query?t2(t.query)||void 0:""),t}function ou(e,t){return t?t(e):nV.of()}function od(e){return"function"==typeof e?new of(e):e}function oh(e){return e.request.length<=1}ew(os,"validateOperation"),ew(ol,"createOperation"),ew(oc,"transformOperation"),ew(ou,"passthrough"),ew(od,"toLink"),ew(oh,"isTerminating");var op=function(e){function t(t,n){var o=e.call(this,t)||this;return o.link=n,o}return eD(t,e),ew(t,"LinkError"),t}(Error),of=function(){function e(e){e&&(this.request=e)}return ew(e,"ApolloLink"),e.empty=function(){return new e(function(){return nV.of()})},e.from=function(t){return 0===t.length?e.empty():t.map(od).reduce(function(e,t){return e.concat(t)})},e.split=function(t,n,o){var r=od(n),i=od(o||new e(ou));return new e(oh(r)&&oh(i)?function(e){return t(e)?r.request(e)||nV.of():i.request(e)||nV.of()}:function(e,n){return t(e)?r.request(e,n)||nV.of():i.request(e,n)||nV.of()})},e.execute=function(e,t){return e.request(ol(t.context,oc(os(t))))||nV.of()},e.concat=function(t,n){var o=od(t);if(oh(o))return __DEV__&&eV.warn(new op("You are calling concat on a terminating link, which will have no effect",o)),o;var r=od(n);return new e(oh(r)?function(e){return o.request(e,function(e){return r.request(e)||nV.of()})||nV.of()}:function(e,t){return o.request(e,function(e){return r.request(e,t)||nV.of()})||nV.of()})},e.prototype.split=function(t,n,o){return this.concat(e.split(t,n,o||new e(ou)))},e.prototype.concat=function(t){return e.concat(this,t)},e.prototype.request=function(e,t){throw __DEV__?new eA("request is not implemented"):new eA(19)},e.prototype.onError=function(e,t){if(t&&t.error)return t.error(e),!1;throw e},e.prototype.setOnError=function(e){return this.onError=e,this},e}(),om=of.execute,og=Object.prototype.hasOwnProperty;function ov(e){return function(t){return t.text().then(function(e){try{return JSON.parse(e)}catch(n){throw n.name="ServerParseError",n.response=t,n.statusCode=t.status,n.bodyText=e,n}}).then(function(n){return t.status>=300&&oa(t,n,"Response not successful: Received status code ".concat(t.status)),Array.isArray(n)||og.call(n,"data")||og.call(n,"errors")||oa(t,n,"Server response was missing for query '".concat(Array.isArray(e)?e.map(function(e){return e.operationName}):e.operationName,"'.")),n})}}ew(ov,"parseAndCheckHttpResponse");var ob=ew(function(e,t){var n;try{n=JSON.stringify(e)}catch(e){var o=__DEV__?new eA("Network request failed. ".concat(t," is not serializable: ").concat(e.message)):new eA(21);throw o.parseError=e,o}return n},"serializeFetchParameter"),oy={http:{includeQuery:!0,includeExtensions:!1},headers:{accept:"*/*","content-type":"application/json"},options:{method:"POST"}},ok=ew(function(e,t){return t(e)},"defaultPrinter");function ow(e,t){for(var n=[],o=2;o<arguments.length;o++)n[o-2]=arguments[o];var r={},i={};n.forEach(function(e){r=eT(eT(eT({},r),e.options),{headers:eT(eT({},r.headers),ox(e.headers))}),e.credentials&&(r.credentials=e.credentials),i=eT(eT({},i),e.http)});var a=e.operationName,s=e.extensions,l=e.variables,c=e.query,u={operationName:a,variables:l};return i.includeExtensions&&(u.extensions=s),i.includeQuery&&(u.query=t(c,tu)),{options:r,body:u}}function ox(e){if(e){var t=Object.create(null);return Object.keys(Object(e)).forEach(function(n){t[n.toLowerCase()]=e[n]}),t}return e}ew(ow,"selectHttpOptionsAndBodyInternal"),ew(ox,"headersToLowerCase");var oS=ew(function(e){if(!e&&"undefined"==typeof fetch)throw __DEV__?new eA(`
"fetch" has not been found globally and no fetcher has been configured. To fix this, install a fetch package (like https://www.npmjs.com/package/cross-fetch), instantiate the fetcher, and pass it into your HttpLink constructor. For example:

import fetch from 'cross-fetch';
import { ApolloClient, HttpLink } from '@apollo/client';
const client = new ApolloClient({
  link: new HttpLink({ uri: '/graphql', fetch })
});
    `):new eA(20)},"checkFetcher"),oC=ew(function(){if("undefined"==typeof AbortController)return{controller:!1,signal:!1};var e=new AbortController,t=e.signal;return{controller:e,signal:t}},"createSignalIfSupported"),oF=ew(function(e,t){return e.getContext().uri||("function"==typeof t?t(e):t||"/graphql")},"selectURI");function o_(e,t){var n=[],o=ew(function(e,t){n.push("".concat(e,"=").concat(encodeURIComponent(t)))},"addQueryParam");if("query"in t&&o("query",t.query),t.operationName&&o("operationName",t.operationName),t.variables){var r=void 0;try{r=ob(t.variables,"Variables map")}catch(e){return{parseError:e}}o("variables",r)}if(t.extensions){var i=void 0;try{i=ob(t.extensions,"Extensions map")}catch(e){return{parseError:e}}o("extensions",i)}var a="",s=e,l=e.indexOf("#");-1!==l&&(a=e.substr(l),s=e.substr(0,l));var c=-1===s.indexOf("?")?"?":"&";return{newURI:s+c+n.join("&")+a}}ew(o_,"rewriteURIForGET");var oN=eH(function(){return fetch}),o$=ew(function(e){void 0===e&&(e={});var t=e.uri,n=void 0===t?"/graphql":t,o=e.fetch,r=e.print,i=void 0===r?ok:r,a=e.includeExtensions,s=e.useGETForQueries,l=e.includeUnusedVariables,c=void 0!==l&&l,u=ez(e,["uri","fetch","print","includeExtensions","useGETForQueries","includeUnusedVariables"]);__DEV__&&oS(o||oN);var d={http:{includeExtensions:a},options:u.fetchOptions,credentials:u.credentials,headers:u.headers};return new of(function(e){var t,r=oF(e,n),a=e.getContext(),l={};if(a.clientAwareness){var u=a.clientAwareness,h=u.name,p=u.version;h&&(l["apollographql-client-name"]=h),p&&(l["apollographql-client-version"]=p)}var f=eT(eT({},l),a.headers),m=ow(e,i,oy,d,{http:a.http,options:a.fetchOptions,credentials:a.credentials,headers:f}),g=m.options,v=m.body;if(v.variables&&!c){var b=new Set(Object.keys(v.variables));tl(e.query,{Variable:function(e,t,n){n&&"VariableDefinition"!==n.kind&&b.delete(e.name.value)}}),b.size&&(v.variables=eT({},v.variables),b.forEach(function(e){delete v.variables[e]}))}if(!g.signal){var y=oC(),k=y.controller,w=y.signal;(t=k)&&(g.signal=w)}var x=ew(function(e){return"OperationDefinition"===e.kind&&"mutation"===e.operation},"definitionIsMutation");if(s&&!e.query.definitions.some(x)&&(g.method="GET"),"GET"===g.method){var S=o_(r,v),C=S.newURI,F=S.parseError;if(F)return oi(F);r=C}else try{g.body=ob(v,"Payload")}catch(e){return oi(e)}return new nV(function(n){return(o||eH(function(){return fetch})||oN)(r,g).then(function(t){return e.setContext({response:t}),t}).then(ov(e)).then(function(e){return n.next(e),n.complete(),e}).catch(function(e){"AbortError"!==e.name&&(e.result&&e.result.errors&&e.result.data&&n.next(e.result),n.error(e))}),function(){t&&t.abort()}})})},"createHttpLink"),oE=function(e){function t(t){void 0===t&&(t={});var n=e.call(this,o$(t).request)||this;return n.options=t,n}return eD(t,e),ew(t,"HttpLink"),t}(of),oO=Object.prototype,oD=oO.toString,oT=oO.hasOwnProperty,oz=Function.prototype.toString,oM=new Map;function oI(e,t){try{return oP(e,t)}finally{oM.clear()}}function oP(e,t){if(e===t)return!0;var n=oD.call(e);if(n!==oD.call(t))return!1;switch(n){case"[object Array]":if(e.length!==t.length)break;case"[object Object]":if(oV(e,t))return!0;var o=oj(e),r=oj(t),i=o.length;if(i!==r.length)break;for(var a=0;a<i;++a)if(!oT.call(t,o[a]))return!1;for(var a=0;a<i;++a){var s=o[a];if(!oP(e[s],t[s]))return!1}return!0;case"[object Error]":return e.name===t.name&&e.message===t.message;case"[object Number]":if(e!=e)return t!=t;case"[object Boolean]":case"[object Date]":return+e==+t;case"[object RegExp]":case"[object String]":return e==""+t;case"[object Map]":case"[object Set]":if(e.size!==t.size)break;if(oV(e,t))return!0;for(var l=e.entries(),c="[object Map]"===n;;){var u=l.next();if(u.done)break;var d=u.value,h=d[0],p=d[1];if(!t.has(h)||c&&!oP(p,t.get(h)))return!1}return!0;case"[object Uint16Array]":case"[object Uint8Array]":case"[object Uint32Array]":case"[object Int32Array]":case"[object Int8Array]":case"[object Int16Array]":case"[object ArrayBuffer]":e=new Uint8Array(e),t=new Uint8Array(t);case"[object DataView]":var f=e.byteLength;if(f===t.byteLength)for(;f--&&e[f]===t[f];);return -1===f;case"[object AsyncFunction]":case"[object GeneratorFunction]":case"[object AsyncGeneratorFunction]":case"[object Function]":var m=oz.call(e);if(m!==oz.call(t))break;return!oA(m,oL)}return!1}function oj(e){return Object.keys(e).filter(oR,e)}function oR(e){return void 0!==this[e]}ew(oI,"equal"),ew(oP,"check"),ew(oj,"definedKeys"),ew(oR,"isDefinedKey");var oL="{ [native code] }";function oA(e,t){var n=e.length-t.length;return n>=0&&e.indexOf(t,n)===n}function oV(e,t){var n=oM.get(e);if(n){if(n.has(t))return!0}else oM.set(e,n=new Set);return n.add(t),!1}ew(oA,"endsWith"),ew(oV,"previouslyCompared");var oq=ew(function(){return Object.create(null)},"defaultMakeData"),oZ=Array.prototype,oB=oZ.forEach,oQ=oZ.slice,oH=function(){function e(e,t){void 0===e&&(e=!0),void 0===t&&(t=oq),this.weakness=e,this.makeData=t}return ew(e,"Trie"),e.prototype.lookup=function(){for(var e=[],t=0;t<arguments.length;t++)e[t]=arguments[t];return this.lookupArray(e)},e.prototype.lookupArray=function(e){var t=this;return oB.call(e,function(e){return t=t.getChildTrie(e)}),t.data||(t.data=this.makeData(oQ.call(e)))},e.prototype.getChildTrie=function(t){var n=this.weakness&&oG(t)?this.weak||(this.weak=new WeakMap):this.strong||(this.strong=new Map),o=n.get(t);return o||n.set(t,o=new e(this.weakness,this.makeData)),o},e}();function oG(e){switch(typeof e){case"object":if(null===e)break;case"function":return!0}return!1}ew(oG,"isObjRef");var oU=null,oW={},oK=1,oY=ew(function(){return function(){function e(){this.id=["slot",oK++,Date.now(),Math.random().toString(36).slice(2)].join(":")}return ew(e,"Slot"),e.prototype.hasValue=function(){for(var e=oU;e;e=e.parent)if(this.id in e.slots){var t=e.slots[this.id];if(t===oW)break;return e!==oU&&(oU.slots[this.id]=t),!0}return oU&&(oU.slots[this.id]=oW),!1},e.prototype.getValue=function(){if(this.hasValue())return oU.slots[this.id]},e.prototype.withValue=function(e,t,n,o){var r,i=((r={__proto__:null})[this.id]=e,r),a=oU;oU={parent:a,slots:i};try{return t.apply(o,n)}finally{oU=a}},e.bind=function(e){var t=oU;return function(){var n=oU;try{return oU=t,e.apply(this,arguments)}finally{oU=n}}},e.noContext=function(e,t,n){if(!oU)return e.apply(n,t);var o=oU;try{return oU=null,e.apply(n,t)}finally{oU=o}},e}()},"makeSlotClass"),oX="@wry/context:Slot",oJ=Array,o0=oJ[oX]||function(){var e=oY();try{Object.defineProperty(oJ,oX,{value:oJ[oX]=e,enumerable:!1,writable:!1,configurable:!1})}finally{return e}}();function o1(){}o0.bind,o0.noContext,ew(o1,"defaultDispose");var o2=function(){function e(e,t){void 0===e&&(e=1/0),void 0===t&&(t=o1),this.max=e,this.dispose=t,this.map=new Map,this.newest=null,this.oldest=null}return ew(e,"Cache"),e.prototype.has=function(e){return this.map.has(e)},e.prototype.get=function(e){var t=this.getNode(e);return t&&t.value},e.prototype.getNode=function(e){var t=this.map.get(e);if(t&&t!==this.newest){var n=t.older,o=t.newer;o&&(o.older=n),n&&(n.newer=o),t.older=this.newest,t.older.newer=t,t.newer=null,this.newest=t,t===this.oldest&&(this.oldest=o)}return t},e.prototype.set=function(e,t){var n=this.getNode(e);return n?n.value=t:(n={key:e,value:t,newer:null,older:this.newest},this.newest&&(this.newest.newer=n),this.newest=n,this.oldest=this.oldest||n,this.map.set(e,n),n.value)},e.prototype.clean=function(){for(;this.oldest&&this.map.size>this.max;)this.delete(this.oldest.key)},e.prototype.delete=function(e){var t=this.map.get(e);return!!t&&(t===this.newest&&(this.newest=t.older),t===this.oldest&&(this.oldest=t.newer),t.newer&&(t.newer.older=t.older),t.older&&(t.older.newer=t.newer),this.map.delete(e),this.dispose(t.value,e),!0)},e}(),o3=new o0,o5=Object.prototype.hasOwnProperty,o4=void 0===(h=Array.from)?function(e){var t=[];return e.forEach(function(e){return t.push(e)}),t}:h;function o6(e){var t=e.unsubscribe;"function"==typeof t&&(e.unsubscribe=void 0,t())}ew(o6,"maybeUnsubscribe");var o8=[];function o9(e,t){if(!e)throw Error(t||"assertion failure")}function o7(e,t){var n=e.length;return n>0&&n===t.length&&e[n-1]===t[n-1]}function re(e){switch(e.length){case 0:throw Error("unknown value");case 1:return e[0];case 2:throw e[1]}}function rt(e){return e.slice(0)}ew(o9,"assert"),ew(o7,"valueIs"),ew(re,"valueGet"),ew(rt,"valueCopy");var rn=function(){function e(t){this.fn=t,this.parents=new Set,this.childValues=new Map,this.dirtyChildren=null,this.dirty=!0,this.recomputing=!1,this.value=[],this.deps=null,++e.count}return ew(e,"Entry"),e.prototype.peek=function(){if(1===this.value.length&&!ra(this))return ro(this),this.value[0]},e.prototype.recompute=function(e){return o9(!this.recomputing,"already recomputing"),ro(this),ra(this)?rr(this,e):re(this.value)},e.prototype.setDirty=function(){this.dirty||(this.dirty=!0,this.value.length=0,rl(this),o6(this))},e.prototype.dispose=function(){var e=this;this.setDirty(),rf(this),ru(this,function(t,n){t.setDirty(),rm(t,e)})},e.prototype.forget=function(){this.dispose()},e.prototype.dependOn=function(e){e.add(this),this.deps||(this.deps=o8.pop()||new Set),this.deps.add(e)},e.prototype.forgetDeps=function(){var e=this;this.deps&&(o4(this.deps).forEach(function(t){return t.delete(e)}),this.deps.clear(),o8.push(this.deps),this.deps=null)},e.count=0,e}();function ro(e){var t=o3.getValue();if(t)return e.parents.add(t),t.childValues.has(e)||t.childValues.set(e,[]),ra(e)?rd(t,e):rh(t,e),t}function rr(e,t){return rf(e),o3.withValue(e,ri,[e,t]),rg(e,t)&&rs(e),re(e.value)}function ri(e,t){e.recomputing=!0,e.value.length=0;try{e.value[0]=e.fn.apply(null,t)}catch(t){e.value[1]=t}e.recomputing=!1}function ra(e){return e.dirty||!!(e.dirtyChildren&&e.dirtyChildren.size)}function rs(e){e.dirty=!1,ra(e)||rc(e)}function rl(e){ru(e,rd)}function rc(e){ru(e,rh)}function ru(e,t){var n=e.parents.size;if(n)for(var o=o4(e.parents),r=0;r<n;++r)t(o[r],e)}function rd(e,t){o9(e.childValues.has(t)),o9(ra(t));var n=!ra(e);if(e.dirtyChildren){if(e.dirtyChildren.has(t))return}else e.dirtyChildren=o8.pop()||new Set;e.dirtyChildren.add(t),n&&rl(e)}function rh(e,t){o9(e.childValues.has(t)),o9(!ra(t));var n=e.childValues.get(t);0===n.length?e.childValues.set(t,rt(t.value)):o7(n,t.value)||e.setDirty(),rp(e,t),ra(e)||rc(e)}function rp(e,t){var n=e.dirtyChildren;n&&(n.delete(t),0===n.size&&(o8.length<100&&o8.push(n),e.dirtyChildren=null))}function rf(e){e.childValues.size>0&&e.childValues.forEach(function(t,n){rm(e,n)}),e.forgetDeps(),o9(null===e.dirtyChildren)}function rm(e,t){t.parents.delete(e),e.childValues.delete(t),rp(e,t)}function rg(e,t){if("function"==typeof e.subscribe)try{o6(e),e.unsubscribe=e.subscribe.apply(null,t)}catch(t){return e.setDirty(),!1}return!0}ew(ro,"rememberParent"),ew(rr,"reallyRecompute"),ew(ri,"recomputeNewValue"),ew(ra,"mightBeDirty"),ew(rs,"setClean"),ew(rl,"reportDirty"),ew(rc,"reportClean"),ew(ru,"eachParent"),ew(rd,"reportDirtyChild"),ew(rh,"reportCleanChild"),ew(rp,"removeDirtyChild"),ew(rf,"forgetChildren"),ew(rm,"forgetChild"),ew(rg,"maybeSubscribe");var rv={setDirty:!0,dispose:!0,forget:!0};function rb(e){var t=new Map,n=e&&e.subscribe;function o(e){var o=o3.getValue();if(o){var r=t.get(e);r||t.set(e,r=new Set),o.dependOn(r),"function"==typeof n&&(o6(r),r.unsubscribe=n(e))}}return ew(o,"depend"),o.dirty=ew(function(e,n){var o=t.get(e);if(o){var r=n&&o5.call(rv,n)?n:"setDirty";o4(o).forEach(function(e){return e[r]()}),t.delete(e),o6(o)}},"dirty"),o}function ry(){var e=new oH("function"==typeof WeakMap);return function(){return e.lookupArray(arguments)}}ew(rb,"dep"),ew(ry,"makeDefaultMakeCacheKeyFunction"),ry();var rk=new Set;function rw(e,t){void 0===t&&(t=Object.create(null));var n=new o2(t.max||65536,function(e){return e.dispose()}),o=t.keyArgs,r=t.makeCacheKey||ry(),i=ew(function(){var i=r.apply(null,o?o.apply(null,arguments):arguments);if(void 0===i)return e.apply(null,arguments);var a=n.get(i);a||(n.set(i,a=new rn(e)),a.subscribe=t.subscribe,a.forget=function(){return n.delete(i)});var s=a.recompute(Array.prototype.slice.call(arguments));return n.set(i,a),rk.add(n),o3.hasValue()||(rk.forEach(function(e){return e.clean()}),rk.clear()),s},"optimistic");function a(e){var t=n.get(e);t&&t.setDirty()}function s(e){var t=n.get(e);if(t)return t.peek()}function l(e){return n.delete(e)}return Object.defineProperty(i,"size",{get:function(){return n.map.size},configurable:!1,enumerable:!1}),ew(a,"dirtyKey"),i.dirtyKey=a,i.dirty=ew(function(){a(r.apply(null,arguments))},"dirty"),ew(s,"peekKey"),i.peekKey=s,i.peek=ew(function(){return s(r.apply(null,arguments))},"peek"),ew(l,"forgetKey"),i.forgetKey=l,i.forget=ew(function(){return l(r.apply(null,arguments))},"forget"),i.makeCacheKey=r,i.getKey=o?ew(function(){return r.apply(null,o.apply(null,arguments))},"getKey"):r,Object.freeze(i)}ew(rw,"wrap");var rx=function(){function e(){this.getFragmentDoc=rw(tN)}return ew(e,"ApolloCache"),e.prototype.batch=function(e){var t,n=this,o="string"==typeof e.optimistic?e.optimistic:!1===e.optimistic?null:void 0;return this.performTransaction(function(){return t=e.update(n)},o),t},e.prototype.recordOptimisticTransaction=function(e,t){this.performTransaction(e,t)},e.prototype.transformDocument=function(e){return e},e.prototype.identify=function(e){},e.prototype.gc=function(){return[]},e.prototype.modify=function(e){return!1},e.prototype.transformForLink=function(e){return e},e.prototype.readQuery=function(e,t){return void 0===t&&(t=!!e.optimistic),this.read(eT(eT({},e),{rootId:e.id||"ROOT_QUERY",optimistic:t}))},e.prototype.readFragment=function(e,t){return void 0===t&&(t=!!e.optimistic),this.read(eT(eT({},e),{query:this.getFragmentDoc(e.fragment,e.fragmentName),rootId:e.id,optimistic:t}))},e.prototype.writeQuery=function(e){var t=e.id,n=e.data,o=ez(e,["id","data"]);return this.write(Object.assign(o,{dataId:t||"ROOT_QUERY",result:n}))},e.prototype.writeFragment=function(e){var t=e.id,n=e.data,o=e.fragment,r=e.fragmentName,i=ez(e,["id","data","fragment","fragmentName"]);return this.write(Object.assign(i,{query:this.getFragmentDoc(o,r),dataId:t,result:n}))},e.prototype.updateQuery=function(e,t){return this.batch({update:function(n){var o=n.readQuery(e),r=t(o);return null==r?o:(n.writeQuery(eT(eT({},e),{data:r})),r)}})},e.prototype.updateFragment=function(e,t){return this.batch({update:function(n){var o=n.readFragment(e),r=t(o);return null==r?o:(n.writeFragment(eT(eT({},e),{data:r})),r)}})},e}(),rS=function(){function e(e,t,n,o){this.message=e,this.path=t,this.query=n,this.variables=o}return ew(e,"MissingFieldError"),e}(),rC=Object.prototype.hasOwnProperty;function rF(e,t){var n=e.__typename,o=e.id,r=e._id;if("string"==typeof n&&(t&&(t.keyObject=void 0!==o?{id:o}:void 0!==r?{_id:r}:void 0),void 0===o&&(o=r),void 0!==o))return"".concat(n,":").concat("number"==typeof o||"string"==typeof o?o:JSON.stringify(o))}ew(rF,"defaultDataIdFromObject");var r_={dataIdFromObject:rF,addTypename:!0,resultCaching:!0,canonizeResults:!1};function rN(e){return oe(r_,e)}function r$(e){var t=e.canonizeResults;return void 0===t?r_.canonizeResults:t}function rE(e,t){return tT(t)?e.get(t.__ref,"__typename"):t&&t.__typename}ew(rN,"normalizeConfig"),ew(r$,"shouldCanonizeResults"),ew(rE,"getTypenameFromStoreObject");var rO=/^[_a-z][_0-9a-z]*/i;function rD(e){var t=e.match(rO);return t?t[0]:e}function rT(e,t,n){return!!tO(t)&&(rI(t)?t.every(function(t){return rT(e,t,n)}):e.selections.every(function(e){if(tX(e)&&tw(e,n)){var o=tK(e);return rC.call(t,o)&&(!e.selectionSet||rT(e.selectionSet,t[o],n))}return!0}))}function rz(e){return tO(e)&&!tT(e)&&!rI(e)}function rM(){return new nv}ew(rD,"fieldNameFromStoreName"),ew(rT,"selectionSetMatchesResult"),ew(rz,"storeValueIsStoreObject"),ew(rM,"makeProcessedFieldsMerger");var rI=ew(function(e){return Array.isArray(e)},"isArray"),rP=Object.create(null),rj=ew(function(){return rP},"delModifier"),rR=Object.create(null),rL=function(){function e(e,t){var n=this;this.policies=e,this.group=t,this.data=Object.create(null),this.rootIds=Object.create(null),this.refs=Object.create(null),this.getFieldValue=function(e,t){return nK(tT(e)?n.get(e.__ref,t):e&&e[t])},this.canRead=function(e){return tT(e)?n.has(e.__ref):"object"==typeof e},this.toReference=function(e,t){if("string"==typeof e)return tD(e);if(tT(e))return e;var o=n.policies.identify(e)[0];if(o){var r=tD(o);return t&&n.merge(o,e),r}}}return ew(e,"EntityStore"),e.prototype.toObject=function(){return eT({},this.data)},e.prototype.has=function(e){return void 0!==this.lookup(e,!0)},e.prototype.get=function(e,t){if(this.group.depend(e,t),rC.call(this.data,e)){var n=this.data[e];if(n&&rC.call(n,t))return n[t]}return"__typename"===t&&rC.call(this.policies.rootTypenamesById,e)?this.policies.rootTypenamesById[e]:this instanceof rZ?this.parent.get(e,t):void 0},e.prototype.lookup=function(e,t){return(t&&this.group.depend(e,"__exists"),rC.call(this.data,e))?this.data[e]:this instanceof rZ?this.parent.lookup(e,t):this.policies.rootTypenamesById[e]?Object.create(null):void 0},e.prototype.merge=function(e,t){var n,o=this;tT(e)&&(e=e.__ref),tT(t)&&(t=t.__ref);var r="string"==typeof e?this.lookup(n=e):e,i="string"==typeof t?this.lookup(n=t):t;if(i){__DEV__?eV("string"==typeof n,"store.merge expects a string ID"):eV("string"==typeof n,1);var a=new nv(rQ).merge(r,i);if(this.data[n]=a,a!==r&&(delete this.refs[n],this.group.caching)){var s=Object.create(null);r||(s.__exists=1),Object.keys(i).forEach(function(e){if(!r||r[e]!==a[e]){s[e]=1;var t=rD(e);t===e||o.policies.hasKeyArgs(a.__typename,t)||(s[t]=1),void 0!==a[e]||o instanceof rZ||delete a[e]}}),s.__typename&&!(r&&r.__typename)&&this.policies.rootTypenamesById[n]===a.__typename&&delete s.__typename,Object.keys(s).forEach(function(e){return o.group.dirty(n,e)})}}},e.prototype.modify=function(e,t){var n=this,o=this.lookup(e);if(o){var r=Object.create(null),i=!1,a=!0,s={DELETE:rP,INVALIDATE:rR,isReference:tT,toReference:this.toReference,canRead:this.canRead,readField:function(t,o){return n.policies.readField("string"==typeof t?{fieldName:t,from:o||tD(e)}:t,{store:n})}};if(Object.keys(o).forEach(function(l){var c=rD(l),u=o[l];if(void 0!==u){var d="function"==typeof t?t:t[l]||t[c];if(d){var h=d===rj?rP:d(nK(u),eT(eT({},s),{fieldName:c,storeFieldName:l,storage:n.getStorage(e,l)}));h===rR?n.group.dirty(e,l):(h===rP&&(h=void 0),h!==u&&(r[l]=h,i=!0,u=h))}void 0!==u&&(a=!1)}}),i)return this.merge(e,r),a&&(this instanceof rZ?this.data[e]=void 0:delete this.data[e],this.group.dirty(e,"__exists")),!0}return!1},e.prototype.delete=function(e,t,n){var o,r=this.lookup(e);if(r){var i=this.getFieldValue(r,"__typename"),a=t&&n?this.policies.getStoreFieldName({typename:i,fieldName:t,args:n}):t;return this.modify(e,a?((o={})[a]=rj,o):rj)}return!1},e.prototype.evict=function(e,t){var n=!1;return e.id&&(rC.call(this.data,e.id)&&(n=this.delete(e.id,e.fieldName,e.args)),this instanceof rZ&&this!==t&&(n=this.parent.evict(e,t)||n),(e.fieldName||n)&&this.group.dirty(e.id,e.fieldName||"__exists")),n},e.prototype.clear=function(){this.replace(null)},e.prototype.extract=function(){var e=this,t=this.toObject(),n=[];return this.getRootIdSet().forEach(function(t){rC.call(e.policies.rootTypenamesById,t)||n.push(t)}),n.length&&(t.__META={extraRootIds:n.sort()}),t},e.prototype.replace=function(e){var t=this;if(Object.keys(this.data).forEach(function(n){e&&rC.call(e,n)||t.delete(n)}),e){var n=e.__META,o=ez(e,["__META"]);Object.keys(o).forEach(function(e){t.merge(e,o[e])}),n&&n.extraRootIds.forEach(this.retain,this)}},e.prototype.retain=function(e){return this.rootIds[e]=(this.rootIds[e]||0)+1},e.prototype.release=function(e){if(this.rootIds[e]>0){var t=--this.rootIds[e];return t||delete this.rootIds[e],t}return 0},e.prototype.getRootIdSet=function(e){return void 0===e&&(e=new Set),Object.keys(this.rootIds).forEach(e.add,e),this instanceof rZ?this.parent.getRootIdSet(e):Object.keys(this.policies.rootTypenamesById).forEach(e.add,e),e},e.prototype.gc=function(){var e=this,t=this.getRootIdSet(),n=this.toObject();t.forEach(function(o){rC.call(n,o)&&(Object.keys(e.findChildRefIds(o)).forEach(t.add,t),delete n[o])});var o=Object.keys(n);if(o.length){for(var r=this;r instanceof rZ;)r=r.parent;o.forEach(function(e){return r.delete(e)})}return o},e.prototype.findChildRefIds=function(e){if(!rC.call(this.refs,e)){var t=this.refs[e]=Object.create(null),n=this.data[e];if(!n)return t;var o=new Set([n]);o.forEach(function(e){tT(e)&&(t[e.__ref]=!0),tO(e)&&Object.keys(e).forEach(function(t){var n=e[t];tO(n)&&o.add(n)})})}return this.refs[e]},e.prototype.makeCacheKey=function(){return this.group.keyMaker.lookupArray(arguments)},e}(),rA=function(){function e(e,t){void 0===t&&(t=null),this.caching=e,this.parent=t,this.d=null,this.resetCaching()}return ew(e,"CacheGroup"),e.prototype.resetCaching=function(){this.d=this.caching?rb():null,this.keyMaker=new oH(nJ)},e.prototype.depend=function(e,t){if(this.d){this.d(rV(e,t));var n=rD(t);n!==t&&this.d(rV(e,n)),this.parent&&this.parent.depend(e,t)}},e.prototype.dirty=function(e,t){this.d&&this.d.dirty(rV(e,t),"__exists"===t?"forget":"setDirty")},e}();function rV(e,t){return t+"#"+e}function rq(e,t){rH(e)&&e.group.depend(t,"__exists")}ew(rV,"makeDepKey"),ew(rq,"maybeDependOnExistenceOfEntity"),i=function(e){function t(t){var n=t.policies,o=t.resultCaching,r=t.seed,i=e.call(this,n,new rA(void 0===o||o))||this;return i.stump=new rB(i),i.storageTrie=new oH(nJ),r&&i.replace(r),i}return eD(t,e),ew(t,"Root"),t.prototype.addLayer=function(e,t){return this.stump.addLayer(e,t)},t.prototype.removeLayer=function(){return this},t.prototype.getStorage=function(){return this.storageTrie.lookupArray(arguments)},t}(r=rL||(rL={})),r.Root=i;var rZ=function(e){function t(t,n,o,r){var i=e.call(this,n.policies,r)||this;return i.id=t,i.parent=n,i.replay=o,i.group=r,o(i),i}return eD(t,e),ew(t,"Layer"),t.prototype.addLayer=function(e,n){return new t(e,this,n,this.group)},t.prototype.removeLayer=function(e){var t=this,n=this.parent.removeLayer(e);return e===this.id?(this.group.caching&&Object.keys(this.data).forEach(function(e){var o=t.data[e],r=n.lookup(e);r?o?o!==r&&Object.keys(o).forEach(function(n){oI(o[n],r[n])||t.group.dirty(e,n)}):(t.group.dirty(e,"__exists"),Object.keys(r).forEach(function(n){t.group.dirty(e,n)})):t.delete(e)}),n):n===this.parent?this:n.addLayer(this.id,this.replay)},t.prototype.toObject=function(){return eT(eT({},this.parent.toObject()),this.data)},t.prototype.findChildRefIds=function(t){var n=this.parent.findChildRefIds(t);return rC.call(this.data,t)?eT(eT({},n),e.prototype.findChildRefIds.call(this,t)):n},t.prototype.getStorage=function(){for(var e=this.parent;e.parent;)e=e.parent;return e.getStorage.apply(e,arguments)},t}(rL),rB=function(e){function t(t){return e.call(this,"EntityStore.Stump",t,function(){},new rA(t.group.caching,t.group))||this}return eD(t,e),ew(t,"Stump"),t.prototype.removeLayer=function(){return this},t.prototype.merge=function(){return this.parent.merge.apply(this.parent,arguments)},t}(rZ);function rQ(e,t,n){var o=e[n],r=t[n];return oI(o,r)?o:r}function rH(e){return!!(e instanceof rL&&e.group.caching)}function rG(e){return tO(e)?rI(e)?e.slice(0):eT({__proto__:Object.getPrototypeOf(e)},e):e}ew(rQ,"storeObjectReconciler"),ew(rH,"supportsResultCaching"),ew(rG,"shallowCopy");var rU=function(){function e(){this.known=new(n0?WeakSet:Set),this.pool=new oH(nJ),this.passes=new WeakMap,this.keysByJSON=new Map,this.empty=this.admit({})}return ew(e,"ObjectCanon"),e.prototype.isKnown=function(e){return tO(e)&&this.known.has(e)},e.prototype.pass=function(e){if(tO(e)){var t=rG(e);return this.passes.set(t,e),t}return e},e.prototype.admit=function(e){var t=this;if(tO(e)){var n=this.passes.get(e);if(n)return n;switch(Object.getPrototypeOf(e)){case Array.prototype:if(this.known.has(e))break;var o=e.map(this.admit,this),r=this.pool.lookupArray(o);return!r.array&&(this.known.add(r.array=o),__DEV__&&Object.freeze(o)),r.array;case null:case Object.prototype:if(this.known.has(e))break;var i=Object.getPrototypeOf(e),a=[i],s=this.sortedKeys(e);a.push(s.json);var l=a.length;s.sorted.forEach(function(n){a.push(t.admit(e[n]))});var r=this.pool.lookupArray(a);if(!r.object){var c=r.object=Object.create(i);this.known.add(c),s.sorted.forEach(function(e,t){c[e]=a[l+t]}),__DEV__&&Object.freeze(c)}return r.object}}return e},e.prototype.sortedKeys=function(e){var t=Object.keys(e),n=this.pool.lookupArray(t);if(!n.keys){t.sort();var o=JSON.stringify(t);(n.keys=this.keysByJSON.get(o))||this.keysByJSON.set(o,n.keys={sorted:t,json:o})}return n.keys},e}(),rW=Object.assign(function(e){if(tO(e)){void 0===p&&rK();var t=p.admit(e),n=f.get(t);return void 0===n&&f.set(t,n=JSON.stringify(t)),n}return JSON.stringify(e)},{reset:rK});function rK(){p=new rU,f=new(nJ?WeakMap:Map)}function rY(e){return[e.selectionSet,e.objectOrReference,e.context,e.context.canonizeResults]}ew(rK,"resetCanonicalStringify"),ew(rY,"execSelectionSetKeyArgs");var rX=function(){function e(e){var t=this;this.knownResults=new(nJ?WeakMap:Map),this.config=oe(e,{addTypename:!1!==e.addTypename,canonizeResults:r$(e)}),this.canon=e.canon||new rU,this.executeSelectionSet=rw(function(e){var n,o=e.context.canonizeResults,r=rY(e);r[3]=!o;var i=(n=t.executeSelectionSet).peek.apply(n,r);return i?o?eT(eT({},i),{result:t.canon.admit(i.result)}):i:(rq(e.context.store,e.enclosingRef.__ref),t.execSelectionSetImpl(e))},{max:this.config.resultCacheMaxSize,keyArgs:rY,makeCacheKey:function(e,t,n,o){if(rH(n.store))return n.store.makeCacheKey(e,tT(t)?t.__ref:t,n.varString,o)}}),this.executeSubSelectedArray=rw(function(e){return rq(e.context.store,e.enclosingRef.__ref),t.execSubSelectedArrayImpl(e)},{max:this.config.resultCacheMaxSize,makeCacheKey:function(e){var t=e.field,n=e.array,o=e.context;if(rH(o.store))return o.store.makeCacheKey(t,n,o.varString)}})}return ew(e,"StoreReader"),e.prototype.resetCanon=function(){this.canon=new rU},e.prototype.diffQueryAgainstStore=function(e){var t,n=e.store,o=e.query,r=e.rootId,i=e.variables,a=e.returnPartialData,s=e.canonizeResults,l=void 0===s?this.config.canonizeResults:s,c=this.config.cache.policies;i=eT(eT({},t8(t5(o))),i);var u=tD(void 0===r?"ROOT_QUERY":r),d=this.executeSelectionSet({selectionSet:t6(o).selectionSet,objectOrReference:u,enclosingRef:u,context:{store:n,query:o,policies:c,variables:i,varString:rW(i),canonizeResults:l,fragmentMap:t$(t3(o))}});if(d.missing&&(t=[new rS(rJ(d.missing),d.missing,o,i)],!(void 0===a||a)))throw t[0];return{result:d.result,complete:!t,missing:t}},e.prototype.isFresh=function(e,t,n,o){if(rH(o.store)&&this.knownResults.get(e)===n){var r=this.executeSelectionSet.peek(n,t,o,this.canon.isKnown(e));if(r&&e===r.result)return!0}return!1},e.prototype.execSelectionSetImpl=function(e){var t,n=this,o=e.selectionSet,r=e.objectOrReference,i=e.enclosingRef,a=e.context;if(tT(r)&&!a.policies.rootTypenamesById[r.__ref]&&!a.store.has(r.__ref))return{result:this.canon.empty,missing:"Dangling reference to missing ".concat(r.__ref," object")};var s=a.variables,l=a.policies,c=a.store.getFieldValue(r,"__typename"),u=[],d=new nv;function h(e,n){var o;return e.missing&&(t=d.merge(t,((o={})[n]=e.missing,o))),e.result}this.config.addTypename&&"string"==typeof c&&!l.rootIdsByTypename[c]&&u.push({__typename:c}),ew(h,"handleMissing");var p=new Set(o.selections);p.forEach(function(e){var o,f;if(tw(e,s)){if(tX(e)){var m=l.readField({fieldName:e.name.value,field:e,variables:a.variables,from:r},a),g=tK(e);void 0===m?nr.added(e)||(t=d.merge(t,((o={})[g]="Can't find field '".concat(e.name.value,"' on ").concat(tT(r)?r.__ref+" object":"object "+JSON.stringify(r,null,2)),o))):rI(m)?m=h(n.executeSubSelectedArray({field:e,array:m,enclosingRef:i,context:a}),g):e.selectionSet?null!=m&&(m=h(n.executeSelectionSet({selectionSet:e.selectionSet,objectOrReference:m,enclosingRef:tT(m)?m:i,context:a}),g)):a.canonizeResults&&(m=n.canon.pass(m)),void 0!==m&&u.push(((f={})[g]=m,f))}else{var v=tE(e,a.fragmentMap);v&&l.fragmentMatches(v,c)&&v.selectionSet.selections.forEach(p.add,p)}}});var f={result:nm(u),missing:t},m=a.canonizeResults?this.canon.admit(f):nK(f);return m.result&&this.knownResults.set(m.result,o),m},e.prototype.execSubSelectedArrayImpl=function(e){var t,n=this,o=e.field,r=e.array,i=e.enclosingRef,a=e.context,s=new nv;function l(e,n){var o;return e.missing&&(t=s.merge(t,((o={})[n]=e.missing,o))),e.result}return ew(l,"handleMissing"),o.selectionSet&&(r=r.filter(a.store.canRead)),r=r.map(function(e,t){return null===e?null:rI(e)?l(n.executeSubSelectedArray({field:o,array:e,enclosingRef:i,context:a}),t):o.selectionSet?l(n.executeSelectionSet({selectionSet:o.selectionSet,objectOrReference:e,enclosingRef:tT(e)?e:i,context:a}),t):(__DEV__&&r0(a.store,o,e),e)}),{result:a.canonizeResults?this.canon.admit(r):r,missing:t}},e}();function rJ(e){try{JSON.stringify(e,function(e,t){if("string"==typeof t)throw t;return t})}catch(e){return e}}function r0(e,t,n){if(!t.selectionSet){var o=new Set([n]);o.forEach(function(n){tO(n)&&(__DEV__?eV(!tT(n),"Missing selection set for object of type ".concat(rE(e,n)," returned for query field ").concat(t.name.value)):eV(!tT(n),5),Object.values(n).forEach(o.add,o))})}}ew(rJ,"firstMissing"),ew(r0,"assertSelectionSetForIdValue");var r1=new o0,r2=new WeakMap;function r3(e){var t=r2.get(e);return t||r2.set(e,t={vars:new Set,dep:rb()}),t}function r5(e){r3(e).vars.forEach(function(t){return t.forgetCache(e)})}function r4(e){r3(e).vars.forEach(function(t){return t.attachCache(e)})}function r6(e){var t=new Set,n=new Set,o=ew(function(i){if(arguments.length>0){if(e!==i){e=i,t.forEach(function(e){r3(e).dep.dirty(o),r8(e)});var a=Array.from(n);n.clear(),a.forEach(function(t){return t(e)})}}else{var s=r1.getValue();s&&(r(s),r3(s).dep(o))}return e},"rv");o.onNextChange=function(e){return n.add(e),function(){n.delete(e)}};var r=o.attachCache=function(e){return t.add(e),r3(e).vars.add(o),o};return o.forgetCache=function(e){return t.delete(e)},o}function r8(e){e.broadcastWatches&&e.broadcastWatches()}ew(r3,"getCacheInfo"),ew(r5,"forgetCache"),ew(r4,"recallCache"),ew(r6,"makeVar"),ew(r8,"broadcast");var r9=Object.create(null);function r7(e){var t=JSON.stringify(e);return r9[t]||(r9[t]=Object.create(null))}function ie(e){var t=r7(e);return t.keyFieldsFn||(t.keyFieldsFn=function(t,n){var o=ew(function(e,t){return n.readField(t,e)},"extract"),r=n.keyObject=io(e,function(e){var r=ia(n.storeObject,e,o);return void 0===r&&t!==n.storeObject&&rC.call(t,e[0])&&(r=ia(t,e,ii)),__DEV__?eV(void 0!==r,"Missing field '".concat(e.join("."),"' while extracting keyFields from ").concat(JSON.stringify(t))):eV(void 0!==r,2),r});return"".concat(n.typename,":").concat(JSON.stringify(r))})}function it(e){var t=r7(e);return t.keyArgsFn||(t.keyArgsFn=function(t,n){var o=n.field,r=n.variables,i=n.fieldName,a=JSON.stringify(io(e,function(e){var n=e[0],i=n.charAt(0);if("@"===i){if(o&&n9(o.directives)){var a=n.slice(1),s=o.directives.find(function(e){return e.name.value===a}),l=s&&tW(s,r);return l&&ia(l,e.slice(1))}return}if("$"===i){var c=n.slice(1);if(r&&rC.call(r,c)){var u=e.slice(0);return u[0]=c,ia(r,u)}return}if(t)return ia(t,e)}));return(t||"{}"!==a)&&(i+=":"+a),i})}function io(e,t){var n=new nv;return ir(e).reduce(function(e,o){var r,i=t(o);if(void 0!==i){for(var a=o.length-1;a>=0;--a)(r={})[o[a]]=i,i=r;e=n.merge(e,i)}return e},Object.create(null))}function ir(e){var t=r7(e);if(!t.paths){var n=t.paths=[],o=[];e.forEach(function(t,r){rI(t)?(ir(t).forEach(function(e){return n.push(o.concat(e))}),o.length=0):(o.push(t),rI(e[r+1])||(n.push(o.slice(0)),o.length=0))})}return t.paths}function ii(e,t){return e[t]}function ia(e,t,n){return n=n||ii,is(t.reduce(ew(function e(t,o){return rI(t)?t.map(function(t){return e(t,o)}):t&&n(t,o)},"reducer"),e))}function is(e){return tO(e)?rI(e)?e.map(is):io(Object.keys(e).sort(),function(t){return ia(e,t)}):e}function il(e){return void 0!==e.args?e.args:e.field?tW(e.field,e.variables):null}ew(r7,"lookupSpecifierInfo"),ew(ie,"keyFieldsFnFromSpecifier"),ew(it,"keyArgsFnFromSpecifier"),ew(io,"collectSpecifierPaths"),ew(ir,"getSpecifierPaths"),ew(ii,"extractKey"),ew(ia,"extractKeyPath"),ew(is,"normalize"),tH.setStringify(rW),ew(il,"argsFromFieldSpecifier");var ic=ew(function(){},"nullKeyFieldsFn"),iu=ew(function(e,t){return t.fieldName},"simpleKeyArgsFn"),id=ew(function(e,t,n){return(0,n.mergeObjects)(e,t)},"mergeTrueFn"),ih=ew(function(e,t){return t},"mergeFalseFn"),ip=function(){function e(e){this.config=e,this.typePolicies=Object.create(null),this.toBeAdded=Object.create(null),this.supertypeMap=new Map,this.fuzzySubtypes=new Map,this.rootIdsByTypename=Object.create(null),this.rootTypenamesById=Object.create(null),this.usingPossibleTypes=!1,this.config=eT({dataIdFromObject:rF},e),this.cache=this.config.cache,this.setRootTypename("Query"),this.setRootTypename("Mutation"),this.setRootTypename("Subscription"),e.possibleTypes&&this.addPossibleTypes(e.possibleTypes),e.typePolicies&&this.addTypePolicies(e.typePolicies)}return ew(e,"Policies"),e.prototype.identify=function(e,t){var n,o,r=this,i=t&&(t.typename||(null===(n=t.storeObject)||void 0===n?void 0:n.__typename))||e.__typename;if(i===this.rootTypenamesById.ROOT_QUERY)return["ROOT_QUERY"];for(var a=t&&t.storeObject||e,s=eT(eT({},t),{typename:i,storeObject:a,readField:t&&t.readField||function(){var e=ig(arguments,a);return r.readField(e,{store:r.cache.data,variables:e.variables})}}),l=i&&this.getTypePolicy(i),c=l&&l.keyFn||this.config.dataIdFromObject;c;){var u=c(e,s);if(rI(u))c=ie(u);else{o=u;break}}return o=o?String(o):void 0,s.keyObject?[o,s.keyObject]:[o]},e.prototype.addTypePolicies=function(e){var t=this;Object.keys(e).forEach(function(n){var o=e[n],r=o.queryType,i=o.mutationType,a=o.subscriptionType,s=ez(o,["queryType","mutationType","subscriptionType"]);r&&t.setRootTypename("Query",n),i&&t.setRootTypename("Mutation",n),a&&t.setRootTypename("Subscription",n),rC.call(t.toBeAdded,n)?t.toBeAdded[n].push(s):t.toBeAdded[n]=[s]})},e.prototype.updateTypePolicy=function(e,t){var n=this,o=this.getTypePolicy(e),r=t.keyFields,i=t.fields;function a(e,t){e.merge="function"==typeof t?t:!0===t?id:!1===t?ih:e.merge}ew(a,"setMerge"),a(o,t.merge),o.keyFn=!1===r?ic:rI(r)?ie(r):"function"==typeof r?r:o.keyFn,i&&Object.keys(i).forEach(function(t){var o=n.getFieldPolicy(e,t,!0),r=i[t];if("function"==typeof r)o.read=r;else{var s=r.keyArgs,l=r.read,c=r.merge;o.keyFn=!1===s?iu:rI(s)?it(s):"function"==typeof s?s:o.keyFn,"function"==typeof l&&(o.read=l),a(o,c)}o.read&&o.merge&&(o.keyFn=o.keyFn||iu)})},e.prototype.setRootTypename=function(e,t){void 0===t&&(t=e);var n="ROOT_"+e.toUpperCase(),o=this.rootTypenamesById[n];t!==o&&(__DEV__?eV(!o||o===e,"Cannot change root ".concat(e," __typename more than once")):eV(!o||o===e,3),o&&delete this.rootIdsByTypename[o],this.rootIdsByTypename[t]=n,this.rootTypenamesById[n]=t)},e.prototype.addPossibleTypes=function(e){var t=this;this.usingPossibleTypes=!0,Object.keys(e).forEach(function(n){t.getSupertypeSet(n,!0),e[n].forEach(function(e){t.getSupertypeSet(e,!0).add(n);var o=e.match(rO);o&&o[0]===e||t.fuzzySubtypes.set(e,new RegExp(e))})})},e.prototype.getTypePolicy=function(e){var t=this;if(!rC.call(this.typePolicies,e)){var n=this.typePolicies[e]=Object.create(null);n.fields=Object.create(null);var o=this.supertypeMap.get(e);o&&o.size&&o.forEach(function(e){var o=t.getTypePolicy(e),r=o.fields;Object.assign(n,ez(o,["fields"])),Object.assign(n.fields,r)})}var r=this.toBeAdded[e];return r&&r.length&&r.splice(0).forEach(function(n){t.updateTypePolicy(e,n)}),this.typePolicies[e]},e.prototype.getFieldPolicy=function(e,t,n){if(e){var o=this.getTypePolicy(e).fields;return o[t]||n&&(o[t]=Object.create(null))}},e.prototype.getSupertypeSet=function(e,t){var n=this.supertypeMap.get(e);return!n&&t&&this.supertypeMap.set(e,n=new Set),n},e.prototype.fragmentMatches=function(e,t,n,o){var r=this;if(!e.typeCondition)return!0;if(!t)return!1;var i=e.typeCondition.name.value;if(t===i)return!0;if(this.usingPossibleTypes&&this.supertypeMap.has(i))for(var a=this.getSupertypeSet(t,!0),s=[a],l=ew(function(e){var t=r.getSupertypeSet(e,!1);t&&t.size&&0>s.indexOf(t)&&s.push(t)},"maybeEnqueue_1"),c=!!(n&&this.fuzzySubtypes.size),u=!1,d=0;d<s.length;++d){var h=s[d];if(h.has(i))return a.has(i)||(u&&__DEV__&&eV.warn("Inferring subtype ".concat(t," of supertype ").concat(i)),a.add(i)),!0;h.forEach(l),c&&d===s.length-1&&rT(e.selectionSet,n,o)&&(c=!1,u=!0,this.fuzzySubtypes.forEach(function(e,n){var o=t.match(e);o&&o[0]===t&&l(n)}))}return!1},e.prototype.hasKeyArgs=function(e,t){var n=this.getFieldPolicy(e,t,!1);return!!(n&&n.keyFn)},e.prototype.getStoreFieldName=function(e){var t,n=e.typename,o=e.fieldName,r=this.getFieldPolicy(n,o,!1),i=r&&r.keyFn;if(i&&n)for(var a={typename:n,fieldName:o,field:e.field||null,variables:e.variables},s=il(e);i;){var l=i(s,a);if(rI(l))i=it(l);else{t=l||o;break}}return(void 0===t&&(t=e.field?tB(e.field,e.variables):tH(o,il(e))),!1===t)?o:o===rD(t)?t:o+":"+t},e.prototype.readField=function(e,t){var n=e.from;if(n&&(e.field||e.fieldName)){if(void 0===e.typename){var o=t.store.getFieldValue(n,"__typename");o&&(e.typename=o)}var r=this.getStoreFieldName(e),i=rD(r),a=t.store.getFieldValue(n,r),s=this.getFieldPolicy(e.typename,i,!1),l=s&&s.read;if(l){var c=im(this,n,e,t,t.store.getStorage(tT(n)?n.__ref:n,r));return r1.withValue(this.cache,l,[a,c])}return a}},e.prototype.getReadFunction=function(e,t){var n=this.getFieldPolicy(e,t,!1);return n&&n.read},e.prototype.getMergeFunction=function(e,t,n){var o=this.getFieldPolicy(e,t,!1),r=o&&o.merge;return!r&&n&&(r=(o=this.getTypePolicy(n))&&o.merge),r},e.prototype.runMergeFunction=function(e,t,n,o,r){var i=n.field,a=n.typename,s=n.merge;return s===id?iv(o.store)(e,t):s===ih?t:(o.overwrite&&(e=void 0),s(e,t,im(this,void 0,{typename:a,fieldName:i.name.value,field:i,variables:o.variables},o,r||Object.create(null))))},e}();function im(e,t,n,o,r){var i=e.getStoreFieldName(n),a=rD(i),s=n.variables||o.variables,l=o.store,c=l.toReference,u=l.canRead;return{args:il(n),field:n.field||null,fieldName:a,storeFieldName:i,variables:s,isReference:tT,toReference:c,storage:r,cache:e.cache,canRead:u,readField:function(){return e.readField(ig(arguments,t,o),o)},mergeObjects:iv(o.store)}}function ig(e,t,n){var o,r=e[0],i=e[1],a=e.length;return"string"==typeof r?o={fieldName:r,from:a>1?i:t}:(o=eT({},r),rC.call(o,"from")||(o.from=t)),__DEV__&&void 0===o.from&&__DEV__&&eV.warn("Undefined 'from' passed to readField with arguments ".concat(oo(Array.from(e)))),void 0===o.variables&&(o.variables=n),o}function iv(e){return ew(function(t,n){if(rI(t)||rI(n))throw __DEV__?new eA("Cannot automatically merge arrays"):new eA(4);if(tO(t)&&tO(n)){var o=e.getFieldValue(t,"__typename"),r=e.getFieldValue(n,"__typename");if(o&&r&&o!==r)return n;if(tT(t)&&rz(n))return e.merge(t.__ref,n),t;if(rz(t)&&tT(n))return e.merge(t,n.__ref),n;if(rz(t)&&rz(n))return eT(eT({},t),n)}return n},"mergeObjects")}function ib(e,t,n){var o="".concat(t).concat(n),r=e.flavors.get(o);return r||e.flavors.set(o,r=e.clientOnly===t&&e.deferred===n?e:eT(eT({},e),{clientOnly:t,deferred:n})),r}ew(im,"makeFieldFunctionOptions"),ew(ig,"normalizeReadFieldOptions"),ew(iv,"makeMergeObjectsFunction"),ew(ib,"getContextFlavor");var iy=function(){function e(e,t){this.cache=e,this.reader=t}return ew(e,"StoreWriter"),e.prototype.writeToStore=function(e,t){var n=this,o=t.query,r=t.result,i=t.dataId,a=t.variables,s=t.overwrite,l=t1(o),c=rM();a=eT(eT({},t8(l)),a);var u={store:e,written:Object.create(null),merge:function(e,t){return c.merge(e,t)},variables:a,varString:rW(a),fragmentMap:t$(t3(o)),overwrite:!!s,incomingById:new Map,clientOnly:!1,deferred:!1,flavors:new Map},d=this.processSelectionSet({result:r||Object.create(null),dataId:i,selectionSet:l.selectionSet,mergeTree:{map:new Map},context:u});if(!tT(d))throw __DEV__?new eA("Could not identify object ".concat(JSON.stringify(r))):new eA(6);return u.incomingById.forEach(function(t,o){var r=t.storeObject,i=t.mergeTree,a=t.fieldNodeSet,s=tD(o);if(i&&i.map.size){var l=n.applyMerges(i,s,r,u);if(tT(l))return;r=l}if(__DEV__&&!u.overwrite){var c=Object.create(null);a.forEach(function(e){e.selectionSet&&(c[e.name.value]=!0)});var d=ew(function(e){return!0===c[rD(e)]},"hasSelectionSet_1"),h=ew(function(e){var t=i&&i.map.get(e);return!!(t&&t.info&&t.info.merge)},"hasMergeFunction_1");Object.keys(r).forEach(function(e){d(e)&&!h(e)&&i_(s,r,e,u.store)})}e.merge(o,r)}),e.retain(d.__ref),d},e.prototype.processSelectionSet=function(e){var t=this,n=e.dataId,o=e.result,r=e.selectionSet,i=e.context,a=e.mergeTree,s=this.cache.policies,l=Object.create(null),c=n&&s.rootTypenamesById[n]||tY(o,r,i.fragmentMap)||n&&i.store.get(n,"__typename");"string"==typeof c&&(l.__typename=c);var u=ew(function(){var e=ig(arguments,l,i.variables);if(tT(e.from)){var t=i.incomingById.get(e.from.__ref);if(t){var n=s.readField(eT(eT({},e),{from:t.storeObject}),i);if(void 0!==n)return n}}return s.readField(e,i)},"readField"),d=new Set;this.flattenFields(r,o,i,c).forEach(function(e,n){var r,i=o[tK(n)];if(d.add(n),void 0!==i){var h=s.getStoreFieldName({typename:c,fieldName:n.name.value,field:n,variables:e.variables}),p=iw(a,h),f=t.processFieldValue(i,n,n.selectionSet?ib(e,!1,!1):e,p),m=void 0;n.selectionSet&&(tT(f)||rz(f))&&(m=u("__typename",f));var g=s.getMergeFunction(c,n.name.value,m);g?p.info={field:n,typename:c,merge:g}:iC(a,h),l=e.merge(l,((r={})[h]=f,r))}else __DEV__&&!e.clientOnly&&!e.deferred&&!nr.added(n)&&!s.getReadFunction(c,n.name.value)&&__DEV__&&eV.error("Missing field '".concat(tK(n),"' while writing result ").concat(JSON.stringify(o,null,2)).substring(0,1e3))});try{var h=s.identify(o,{typename:c,selectionSet:r,fragmentMap:i.fragmentMap,storeObject:l,readField:u}),p=h[0],f=h[1];n=n||p,f&&(l=i.merge(l,f))}catch(e){if(!n)throw e}if("string"==typeof n){var m=tD(n),g=i.written[n]||(i.written[n]=[]);if(g.indexOf(r)>=0||(g.push(r),this.reader&&this.reader.isFresh(o,m,r,i)))return m;var v=i.incomingById.get(n);return v?(v.storeObject=i.merge(v.storeObject,l),v.mergeTree=ix(v.mergeTree,a),d.forEach(function(e){return v.fieldNodeSet.add(e)})):i.incomingById.set(n,{storeObject:l,mergeTree:iS(a)?void 0:a,fieldNodeSet:d}),m}return l},e.prototype.processFieldValue=function(e,t,n,o){var r=this;return t.selectionSet&&null!==e?rI(e)?e.map(function(e,i){var a=r.processFieldValue(e,t,n,iw(o,i));return iC(o,i),a}):this.processSelectionSet({result:e,selectionSet:t.selectionSet,context:n,mergeTree:o}):__DEV__?nH(e):e},e.prototype.flattenFields=function(e,t,n,o){void 0===o&&(o=tY(t,e,n.fragmentMap));var r=new Map;return this.cache.policies,new oH(!1),r},e.prototype.applyMerges=function(e,t,n,o,r){var i=this;if(e.map.size&&!tT(n)){var a,s,l=!rI(n)&&(tT(t)||rz(t))?t:void 0,c=n;l&&!r&&(r=[tT(l)?l.__ref:l]);var u=ew(function(e,t){return rI(e)?"number"==typeof t?e[t]:void 0:o.store.getFieldValue(e,String(t))},"getValue_1");e.map.forEach(function(e,t){var n=u(l,t),a=u(c,t);if(void 0!==a){r&&r.push(t);var d=i.applyMerges(e,n,a,o,r);d!==a&&(s=s||new Map).set(t,d),r&&eV(r.pop()===t)}}),s&&(n=rI(c)?c.slice(0):eT({},c),s.forEach(function(e,t){n[t]=e}))}return e.info?this.cache.policies.runMergeFunction(t,n,e.info,o,r&&(a=o.store).getStorage.apply(a,r)):n},e}(),ik=[];function iw(e,t){var n=e.map;return n.has(t)||n.set(t,ik.pop()||{map:new Map}),n.get(t)}function ix(e,t){if(e===t||!t||iS(t))return e;if(!e||iS(e))return t;var n=e.info&&t.info?eT(eT({},e.info),t.info):e.info||t.info,o=e.map.size&&t.map.size,r={info:n,map:o?new Map:e.map.size?e.map:t.map};if(o){var i=new Set(t.map.keys());e.map.forEach(function(e,n){r.map.set(n,ix(e,t.map.get(n))),i.delete(n)}),i.forEach(function(n){r.map.set(n,ix(t.map.get(n),e.map.get(n)))})}return r}function iS(e){return!e||!(e.info||e.map.size)}function iC(e,t){var n=e.map,o=n.get(t);o&&iS(o)&&(ik.push(o),n.delete(t))}ew(iw,"getChildMergeTree"),ew(ix,"mergeMergeTrees"),ew(iS,"mergeTreeIsEmpty"),ew(iC,"maybeRecycleChildMergeTree");var iF=new Set;function i_(e,t,n,o){var r=ew(function(e){var t=o.getFieldValue(e,n);return"object"==typeof t&&t},"getChild"),i=r(e);if(i){var a=r(t);if(!(!a||tT(i)||oI(i,a)||Object.keys(i).every(function(e){return void 0!==o.getFieldValue(a,e)}))){var s=o.getFieldValue(e,"__typename")||o.getFieldValue(t,"__typename"),l=rD(n),c="".concat(s,".").concat(l);if(!iF.has(c)){iF.add(c);var u=[];rI(i)||rI(a)||[i,a].forEach(function(e){var t=o.getFieldValue(e,"__typename");"string"!=typeof t||u.includes(t)||u.push(t)}),__DEV__&&eV.warn("Cache data may be lost when replacing the ".concat(l," field of a ").concat(s," object.\n\nTo address this problem (which is not a bug in Apollo Client), ").concat(u.length?"either ensure all objects of type "+u.join(" and ")+" have an ID or a custom merge function, or ":"","define a custom merge function for the ").concat(c," field, so InMemoryCache can safely merge these objects:\n\n  existing: ").concat(JSON.stringify(i).slice(0,1e3),"\n  incoming: ").concat(JSON.stringify(a).slice(0,1e3),"\n\nFor more information about these options, please refer to the documentation:\n\n  * Ensuring entity objects have IDs: https://go.apollo.dev/c/generating-unique-identifiers\n  * Defining custom merge functions: https://go.apollo.dev/c/merging-non-normalized-objects\n"))}}}}ew(i_,"warnAboutDataLoss");var iN=function(e){function t(t){void 0===t&&(t={});var n=e.call(this)||this;return n.watches=new Set,n.typenameDocumentCache=new Map,n.makeVar=r6,n.txCount=0,n.config=rN(t),n.addTypename=!!n.config.addTypename,n.policies=new ip({cache:n,dataIdFromObject:n.config.dataIdFromObject,possibleTypes:n.config.possibleTypes,typePolicies:n.config.typePolicies}),n.init(),n}return eD(t,e),ew(t,"InMemoryCache"),t.prototype.init=function(){var e=this.data=new rL.Root({policies:this.policies,resultCaching:this.config.resultCaching});this.optimisticData=e.stump,this.resetResultCache()},t.prototype.resetResultCache=function(e){var t=this,n=this.storeReader;this.storeWriter=new iy(this,this.storeReader=new rX({cache:this,addTypename:this.addTypename,resultCacheMaxSize:this.config.resultCacheMaxSize,canonizeResults:r$(this.config),canon:e?void 0:n&&n.canon})),this.maybeBroadcastWatch=rw(function(e,n){return t.broadcastWatch(e,n)},{max:this.config.resultCacheMaxSize,makeCacheKey:function(e){var n=e.optimistic?t.optimisticData:t.data;if(rH(n)){var o=e.optimistic,r=e.rootId,i=e.variables;return n.makeCacheKey(e.query,e.callback,rW({optimistic:o,rootId:r,variables:i}))}}}),new Set([this.data.group,this.optimisticData.group]).forEach(function(e){return e.resetCaching()})},t.prototype.restore=function(e){return this.init(),e&&this.data.replace(e),this},t.prototype.extract=function(e){return void 0===e&&(e=!1),(e?this.optimisticData:this.data).extract()},t.prototype.read=function(e){var t=e.returnPartialData;try{return this.storeReader.diffQueryAgainstStore(eT(eT({},e),{store:e.optimistic?this.optimisticData:this.data,config:this.config,returnPartialData:void 0!==t&&t})).result||null}catch(e){if(e instanceof rS)return null;throw e}},t.prototype.write=function(e){try{return++this.txCount,this.storeWriter.writeToStore(this.data,e)}finally{--this.txCount||!1===e.broadcast||this.broadcastWatches()}},t.prototype.modify=function(e){if(rC.call(e,"id")&&!e.id)return!1;var t=e.optimistic?this.optimisticData:this.data;try{return++this.txCount,t.modify(e.id||"ROOT_QUERY",e.fields)}finally{--this.txCount||!1===e.broadcast||this.broadcastWatches()}},t.prototype.diff=function(e){return this.storeReader.diffQueryAgainstStore(eT(eT({},e),{store:e.optimistic?this.optimisticData:this.data,rootId:e.id||"ROOT_QUERY",config:this.config}))},t.prototype.watch=function(e){var t=this;return this.watches.size||r4(this),this.watches.add(e),e.immediate&&this.maybeBroadcastWatch(e),function(){t.watches.delete(e)&&!t.watches.size&&r5(t),t.maybeBroadcastWatch.forget(e)}},t.prototype.gc=function(e){rW.reset();var t=this.optimisticData.gc();return e&&!this.txCount&&(e.resetResultCache?this.resetResultCache(e.resetResultIdentities):e.resetResultIdentities&&this.storeReader.resetCanon()),t},t.prototype.retain=function(e,t){return(t?this.optimisticData:this.data).retain(e)},t.prototype.release=function(e,t){return(t?this.optimisticData:this.data).release(e)},t.prototype.identify=function(e){if(tT(e))return e.__ref;try{return this.policies.identify(e)[0]}catch(e){__DEV__&&eV.warn(e)}},t.prototype.evict=function(e){if(!e.id){if(rC.call(e,"id"))return!1;e=eT(eT({},e),{id:"ROOT_QUERY"})}try{return++this.txCount,this.optimisticData.evict(e,this.data)}finally{--this.txCount||!1===e.broadcast||this.broadcastWatches()}},t.prototype.reset=function(e){var t=this;return this.init(),rW.reset(),e&&e.discardWatches?(this.watches.forEach(function(e){return t.maybeBroadcastWatch.forget(e)}),this.watches.clear(),r5(this)):this.broadcastWatches(),Promise.resolve()},t.prototype.removeOptimistic=function(e){var t=this.optimisticData.removeLayer(e);t!==this.optimisticData&&(this.optimisticData=t,this.broadcastWatches())},t.prototype.batch=function(e){var t,n=this,o=e.update,r=e.optimistic,i=void 0===r||r,a=e.removeOptimistic,s=e.onWatchUpdated,l=ew(function(e){var r=n.data,i=n.optimisticData;++n.txCount,e&&(n.data=n.optimisticData=e);try{return t=o(n)}finally{--n.txCount,n.data=r,n.optimisticData=i}},"perform"),c=new Set;return s&&!this.txCount&&this.broadcastWatches(eT(eT({},e),{onWatchUpdated:function(e){return c.add(e),!1}})),"string"==typeof i?this.optimisticData=this.optimisticData.addLayer(i,l):!1===i?l(this.data):l(),"string"==typeof a&&(this.optimisticData=this.optimisticData.removeLayer(a)),s&&c.size?(this.broadcastWatches(eT(eT({},e),{onWatchUpdated:function(e,t){var n=s.call(this,e,t);return!1!==n&&c.delete(e),n}})),c.size&&c.forEach(function(e){return n.maybeBroadcastWatch.dirty(e)})):this.broadcastWatches(e),t},t.prototype.performTransaction=function(e,t){return this.batch({update:e,optimistic:t||null!==t})},t.prototype.transformDocument=function(e){if(this.addTypename){var t=this.typenameDocumentCache.get(e);return t||(t=nr(e),this.typenameDocumentCache.set(e,t),this.typenameDocumentCache.set(t,t)),t}return e},t.prototype.broadcastWatches=function(e){var t=this;this.txCount||this.watches.forEach(function(n){return t.maybeBroadcastWatch(n,e)})},t.prototype.broadcastWatch=function(e,t){var n=e.lastDiff,o=this.diff(e);(!t||(e.optimistic&&"string"==typeof t.optimistic&&(o.fromOptimisticTransaction=!0),!t.onWatchUpdated||!1!==t.onWatchUpdated.call(this,e,o,n)))&&(n&&oI(n.result,o.result)||e.callback(e.lastDiff=o,n))},t}(rx);function i$(e){return e.hasOwnProperty("graphQLErrors")}ew(i$,"isApolloError");var iE=ew(function(e){var t="";return(n9(e.graphQLErrors)||n9(e.clientErrors))&&(e.graphQLErrors||[]).concat(e.clientErrors||[]).forEach(function(e){var n=e?e.message:"Error message not found.";t+="".concat(n,"\n")}),e.networkError&&(t+="".concat(e.networkError.message,"\n")),t=t.replace(/\n$/,"")},"generateErrorMessage"),iO=function(e){function t(n){var o=n.graphQLErrors,r=n.clientErrors,i=n.networkError,a=n.errorMessage,s=n.extraInfo,l=e.call(this,a)||this;return l.graphQLErrors=o||[],l.clientErrors=r||[],l.networkError=i||null,l.message=a||iE(l),l.extraInfo=s,l.__proto__=t.prototype,l}return eD(t,e),ew(t,"ApolloError"),t}(Error);function iD(e){return!!e&&e<7}(a=m||(m={}))[a.loading=1]="loading",a[a.setVariables=2]="setVariables",a[a.fetchMore=3]="fetchMore",a[a.refetch=4]="refetch",a[a.poll=6]="poll",a[a.ready=7]="ready",a[a.error=8]="error",ew(iD,"isNetworkRequestInFlight");var iT=Object.assign,iz=Object.hasOwnProperty,iM=function(e){function t(t){var n=t.queryManager,o=t.queryInfo,r=t.options,i=e.call(this,function(e){try{var t=e._subscription._observer;t&&!t.error&&(t.error=iP)}catch(e){}var n=!i.observers.size;i.observers.add(e);var o=i.last;return o&&o.error?e.error&&e.error(o.error):o&&o.result&&e.next&&e.next(o.result),n&&i.reobserve().catch(function(){}),function(){i.observers.delete(e)&&!i.observers.size&&i.tearDownQuery()}})||this;i.observers=new Set,i.subscriptions=new Set,i.queryInfo=o,i.queryManager=n,i.isTornDown=!1;var a=n.defaultOptions.watchQuery,s=(void 0===a?{}:a).fetchPolicy,l=void 0===s?"cache-first":s,c=r.fetchPolicy,u=void 0===c?l:c,d=r.initialFetchPolicy,h=void 0===d?"standby"===u?l:u:d;i.options=eT(eT({},r),{initialFetchPolicy:h,fetchPolicy:u}),i.queryId=o.queryId||n.generateQueryId();var p=t1(i.query);return i.queryName=p&&p.name&&p.name.value,i}return eD(t,e),ew(t,"ObservableQuery"),Object.defineProperty(t.prototype,"query",{get:function(){return this.queryManager.transform(this.options.query).document},enumerable:!1,configurable:!0}),Object.defineProperty(t.prototype,"variables",{get:function(){return this.options.variables},enumerable:!1,configurable:!0}),t.prototype.result=function(){var e=this;return new Promise(function(t,n){var o={next:function(n){t(n),e.observers.delete(o),e.observers.size||e.queryManager.removeQuery(e.queryId),setTimeout(function(){r.unsubscribe()},0)},error:n},r=e.subscribe(o)})},t.prototype.getCurrentResult=function(e){void 0===e&&(e=!0);var t=this.getLastResult(!0),n=this.queryInfo.networkStatus||t&&t.networkStatus||m.ready,o=eT(eT({},t),{loading:iD(n),networkStatus:n}),r=this.options.fetchPolicy,i=void 0===r?"cache-first":r;if("network-only"===i||"no-cache"===i||"standby"===i||this.queryManager.transform(this.options.query).hasForcedResolvers);else{var a=this.queryInfo.getDiff();(a.complete||this.options.returnPartialData)&&(o.data=a.result),oI(o.data,{})&&(o.data=void 0),a.complete?(delete o.partial,a.complete&&o.networkStatus===m.loading&&("cache-first"===i||"cache-only"===i)&&(o.networkStatus=m.ready,o.loading=!1)):o.partial=!0,!__DEV__||a.complete||this.options.partialRefetch||o.loading||o.data||o.error||ij(a.missing)}return e&&this.updateLastResult(o),o},t.prototype.isDifferentFromLastResult=function(e){return!this.last||!oI(this.last.result,e)},t.prototype.getLast=function(e,t){var n=this.last;if(n&&n[e]&&(!t||oI(n.variables,this.variables)))return n[e]},t.prototype.getLastResult=function(e){return this.getLast("result",e)},t.prototype.getLastError=function(e){return this.getLast("error",e)},t.prototype.resetLastResults=function(){delete this.last,this.isTornDown=!1},t.prototype.resetQueryStoreErrors=function(){this.queryManager.resetErrors(this.queryId)},t.prototype.refetch=function(e){var t,n={pollInterval:0},o=this.options.fetchPolicy;if("cache-and-network"===o?n.fetchPolicy=o:"no-cache"===o?n.fetchPolicy="no-cache":n.fetchPolicy="network-only",__DEV__&&e&&iz.call(e,"variables")){var r=t5(this.query),i=r.variableDefinitions;(!i||!i.some(function(e){return"variables"===e.variable.name.value}))&&__DEV__&&eV.warn("Called refetch(".concat(JSON.stringify(e),") for query ").concat((null===(t=r.name)||void 0===t?void 0:t.value)||JSON.stringify(r),", which does not declare a $variables variable.\nDid you mean to call refetch(variables) instead of refetch({ variables })?"))}return e&&!oI(this.options.variables,e)&&(n.variables=this.options.variables=eT(eT({},this.options.variables),e)),this.queryInfo.resetLastWrite(),this.reobserve(n,m.refetch)},t.prototype.fetchMore=function(e){var t=this,n=eT(eT({},e.query?e:eT(eT(eT(eT({},this.options),{query:this.query}),e),{variables:eT(eT({},this.options.variables),e.variables)})),{fetchPolicy:"no-cache"}),o=this.queryManager.generateQueryId(),r=this.queryInfo,i=r.networkStatus;r.networkStatus=m.fetchMore,n.notifyOnNetworkStatusChange&&this.observe();var a=new Set;return this.queryManager.fetchQuery(o,n,m.fetchMore).then(function(s){return t.queryManager.removeQuery(o),r.networkStatus===m.fetchMore&&(r.networkStatus=i),t.queryManager.cache.batch({update:function(o){var r=e.updateQuery;r?o.updateQuery({query:t.query,variables:t.variables,returnPartialData:!0,optimistic:!1},function(e){return r(e,{fetchMoreResult:s.data,variables:n.variables})}):o.writeQuery({query:n.query,variables:n.variables,data:s.data})},onWatchUpdated:function(e){a.add(e.query)}}),s}).finally(function(){a.has(t.query)||iI(t)})},t.prototype.subscribeToMore=function(e){var t=this,n=this.queryManager.startGraphQLSubscription({query:e.document,variables:e.variables,context:e.context}).subscribe({next:function(n){var o=e.updateQuery;o&&t.updateQuery(function(e,t){return o(e,{subscriptionData:n,variables:t.variables})})},error:function(t){if(e.onError){e.onError(t);return}__DEV__&&eV.error("Unhandled GraphQL subscription error",t)}});return this.subscriptions.add(n),function(){t.subscriptions.delete(n)&&n.unsubscribe()}},t.prototype.setOptions=function(e){return this.reobserve(e)},t.prototype.setVariables=function(e){return oI(this.variables,e)?this.observers.size?this.result():Promise.resolve():(this.options.variables=e,this.observers.size)?this.reobserve({fetchPolicy:this.options.initialFetchPolicy,variables:e},m.setVariables):Promise.resolve()},t.prototype.updateQuery=function(e){var t=this.queryManager,n=e(t.cache.diff({query:this.options.query,variables:this.variables,returnPartialData:!0,optimistic:!1}).result,{variables:this.variables});n&&(t.cache.writeQuery({query:this.options.query,data:n,variables:this.variables}),t.broadcastQueries())},t.prototype.startPolling=function(e){this.options.pollInterval=e,this.updatePolling()},t.prototype.stopPolling=function(){this.options.pollInterval=0,this.updatePolling()},t.prototype.applyNextFetchPolicy=function(e,t){if(t.nextFetchPolicy){var n=t.fetchPolicy,o=void 0===n?"cache-first":n,r=t.initialFetchPolicy,i=void 0===r?o:r;"function"==typeof t.nextFetchPolicy?t.fetchPolicy=t.nextFetchPolicy(o,{reason:e,options:t,observable:this,initialFetchPolicy:i}):"variables-changed"===e?t.fetchPolicy=i:t.fetchPolicy=t.nextFetchPolicy}return t.fetchPolicy},t.prototype.fetch=function(e,t){return this.queryManager.setObservableQuery(this),this.queryManager.fetchQueryObservable(this.queryId,e,t)},t.prototype.updatePolling=function(){var e=this;if(!this.queryManager.ssrMode){var t=this.pollingInfo,n=this.options.pollInterval;if(!n){t&&(clearTimeout(t.timeout),delete this.pollingInfo);return}if(!t||t.interval!==n){__DEV__?eV(n,"Attempted to start a polling query without a polling interval."):eV(n,10),(t||(this.pollingInfo={})).interval=n;var o=ew(function(){e.pollingInfo&&(iD(e.queryInfo.networkStatus)?r():e.reobserve({fetchPolicy:"network-only"},m.poll).then(r,r))},"maybeFetch"),r=ew(function(){var t=e.pollingInfo;t&&(clearTimeout(t.timeout),t.timeout=setTimeout(o,t.interval))},"poll");r()}}},t.prototype.updateLastResult=function(e,t){return void 0===t&&(t=this.variables),this.last=eT(eT({},this.last),{result:this.queryManager.assumeImmutableResults?e:nH(e),variables:t}),n9(e.errors)||delete this.last.error,this.last},t.prototype.reobserve=function(e,t){var n=this;this.isTornDown=!1;var o=t===m.refetch||t===m.fetchMore||t===m.poll,r=this.options.variables,i=this.options.fetchPolicy,a=oe(this.options,e||{}),s=o?a:iT(this.options,a);o||(this.updatePolling(),!e||!e.variables||oI(e.variables,r)||e.fetchPolicy&&e.fetchPolicy!==i||(this.applyNextFetchPolicy("variables-changed",s),void 0!==t||(t=m.setVariables)));var l=s.variables&&eT({},s.variables),c=this.fetch(s,t),u={next:function(e){n.reportResult(e,l)},error:function(e){n.reportError(e,l)}};return o||(this.concast&&this.observer&&this.concast.removeObserver(this.observer,!0),this.concast=c,this.observer=u),c.addObserver(u),c.promise},t.prototype.observe=function(){this.reportResult(this.getCurrentResult(!1),this.variables)},t.prototype.reportResult=function(e,t){var n=this.getLastError();(n||this.isDifferentFromLastResult(e))&&((n||!e.partial||this.options.returnPartialData)&&this.updateLastResult(e,t),nY(this.observers,"next",e))},t.prototype.reportError=function(e,t){var n=eT(eT({},this.getLastResult()),{error:e,errors:e.graphQLErrors,networkStatus:m.error,loading:!1});this.updateLastResult(n,t),nY(this.observers,"error",this.last.error=e)},t.prototype.hasObservers=function(){return this.observers.size>0},t.prototype.tearDownQuery=function(){this.isTornDown||(this.concast&&this.observer&&(this.concast.removeObserver(this.observer),delete this.concast,delete this.observer),this.stopPolling(),this.subscriptions.forEach(function(e){return e.unsubscribe()}),this.subscriptions.clear(),this.queryManager.stopQuery(this.queryId),this.observers.clear(),this.isTornDown=!0)},t}(nV);function iI(e){var t=e.options,n=t.fetchPolicy,o=t.nextFetchPolicy;return"cache-and-network"===n||"network-only"===n?e.reobserve({fetchPolicy:"cache-first",nextFetchPolicy:function(){return(this.nextFetchPolicy=o,"function"==typeof o)?o.apply(this,arguments):n}}):e.reobserve()}function iP(e){__DEV__&&eV.error("Unhandled error",e.message,e.stack)}function ij(e){__DEV__&&e&&__DEV__&&eV.debug("Missing cache result fields: ".concat(JSON.stringify(e)),e)}n4(iM),ew(iI,"reobserveCacheFirst"),ew(iP,"defaultSubscriptionObserverErrorCallback"),ew(ij,"logMissingFieldErrors");var iR=function(){function e(e){var t=e.cache,n=e.client,o=e.resolvers,r=e.fragmentMatcher;this.cache=t,n&&(this.client=n),o&&this.addResolvers(o),r&&this.setFragmentMatcher(r)}return ew(e,"LocalState"),e.prototype.addResolvers=function(e){var t=this;this.resolvers=this.resolvers||{},Array.isArray(e)?e.forEach(function(e){t.resolvers=nf(t.resolvers,e)}):this.resolvers=nf(this.resolvers,e)},e.prototype.setResolvers=function(e){this.resolvers={},this.addResolvers(e)},e.prototype.getResolvers=function(){return this.resolvers||{}},e.prototype.runResolvers=function(e){var t=e.document,n=e.remoteResult,o=e.context,r=e.variables,i=e.onlyRunForcedResolvers,a=void 0!==i&&i;return eM(this,void 0,void 0,function(){return eI(this,function(e){return t?[2,this.resolveDocument(t,n.data,o,r,this.fragmentMatcher,a).then(function(e){return eT(eT({},n),{data:e.result})})]:[2,n]})})},e.prototype.setFragmentMatcher=function(e){this.fragmentMatcher=e},e.prototype.getFragmentMatcher=function(){return this.fragmentMatcher},e.prototype.clientQuery=function(e){return tS(["client"],e)&&this.resolvers?e:null},e.prototype.serverQuery=function(e){return nh(e)},e.prototype.prepareContext=function(e){var t=this.cache;return eT(eT({},e),{cache:t,getCacheKey:function(e){return t.identify(e)}})},e.prototype.addExportedVariables=function(e,t,n){return void 0===t&&(t={}),void 0===n&&(n={}),eM(this,void 0,void 0,function(){return eI(this,function(o){return e?[2,this.resolveDocument(e,this.buildRootValueFromCache(e,t)||{},this.prepareContext(n),t).then(function(e){return eT(eT({},t),e.exportedVariables)})]:[2,eT({},t)]})})},e.prototype.shouldForceResolvers=function(e){var t=!1;return tl(e,{Directive:{enter:function(e){if("client"===e.name.value&&e.arguments&&(t=e.arguments.some(function(e){return"always"===e.name.value&&"BooleanValue"===e.value.kind&&!0===e.value.value})))return ts}}}),t},e.prototype.buildRootValueFromCache=function(e,t){return this.cache.diff({query:nd(e),variables:t,returnPartialData:!0,optimistic:!1}).result},e.prototype.resolveDocument=function(e,t,n,o,r,i){return void 0===n&&(n={}),void 0===o&&(o={}),void 0===r&&(r=ew(function(){return!0},"fragmentMatcher")),void 0===i&&(i=!1),eM(this,void 0,void 0,function(){var a,s,l,c,u,d,h,p;return eI(this,function(f){return a=t6(e),s=t$(t3(e)),c=(l=a.operation)?l.charAt(0).toUpperCase()+l.slice(1):"Query",u=this,d=u.cache,h=u.client,p={fragmentMap:s,context:eT(eT({},n),{cache:d,client:h}),variables:o,fragmentMatcher:r,defaultOperationType:c,exportedVariables:{},onlyRunForcedResolvers:i},[2,this.resolveSelectionSet(a.selectionSet,t,p).then(function(e){return{result:e,exportedVariables:p.exportedVariables}})]})})},e.prototype.resolveSelectionSet=function(e,t,n){return eM(this,void 0,void 0,function(){var o,r,i,a,s,l=this;return eI(this,function(c){return o=n.fragmentMap,r=n.context,i=n.variables,a=[t],s=ew(function(e){return eM(l,void 0,void 0,function(){var s,l;return eI(this,function(c){return tw(e,i)?tX(e)?[2,this.resolveField(e,t,n).then(function(t){var n;void 0!==t&&a.push(((n={})[tK(e)]=t,n))})]:(tJ(e)?s=e:(s=o[e.name.value],__DEV__?eV(s,"No fragment named ".concat(e.name.value)):eV(s,9)),s&&s.typeCondition&&(l=s.typeCondition.name.value,n.fragmentMatcher(t,l,r)))?[2,this.resolveSelectionSet(s.selectionSet,t,n).then(function(e){a.push(e)})]:[2]:[2]})})},"execute"),[2,Promise.all(e.selections.map(s)).then(function(){return nm(a)})]})})},e.prototype.resolveField=function(e,t,n){return eM(this,void 0,void 0,function(){var o,r,i,a,s,l,c,u,d,h=this;return eI(this,function(p){return o=n.variables,a=(r=e.name.value)!==(i=tK(e)),l=Promise.resolve(s=t[i]||t[r]),(!n.onlyRunForcedResolvers||this.shouldForceResolvers(e))&&(c=t.__typename||n.defaultOperationType,(u=this.resolvers&&this.resolvers[c])&&(d=u[a?r:i])&&(l=Promise.resolve(r1.withValue(this.cache,d,[t,tW(e,o),n.context,{field:e,fragmentMap:n.fragmentMap}])))),[2,l.then(function(t){return(void 0===t&&(t=s),e.directives&&e.directives.forEach(function(e){"export"===e.name.value&&e.arguments&&e.arguments.forEach(function(e){"as"===e.name.value&&"StringValue"===e.value.kind&&(n.exportedVariables[e.value.value]=t)})}),e.selectionSet&&null!=t)?Array.isArray(t)?h.resolveSubSelectedArray(e,t,n):e.selectionSet?h.resolveSelectionSet(e.selectionSet,t,n):void 0:t})]})})},e.prototype.resolveSubSelectedArray=function(e,t,n){var o=this;return Promise.all(t.map(function(t){return null===t?null:Array.isArray(t)?o.resolveSubSelectedArray(e,t,n):e.selectionSet?o.resolveSelectionSet(e.selectionSet,t,n):void 0}))},e}(),iL=new(nJ?WeakMap:Map);function iA(e,t){var n=e[t];"function"==typeof n&&(e[t]=function(){return iL.set(e,(iL.get(e)+1)%1e15),n.apply(this,arguments)})}function iV(e){e.notifyTimeout&&(clearTimeout(e.notifyTimeout),e.notifyTimeout=void 0)}ew(iA,"wrapDestructiveCacheMethod"),ew(iV,"cancelNotifyTimeout");var iq=function(){function e(e,t){void 0===t&&(t=e.generateQueryId()),this.queryId=t,this.listeners=new Set,this.document=null,this.lastRequestId=1,this.subscriptions=new Set,this.stopped=!1,this.dirty=!1,this.observableQuery=null;var n=this.cache=e.cache;iL.has(n)||(iL.set(n,0),iA(n,"evict"),iA(n,"modify"),iA(n,"reset"))}return ew(e,"QueryInfo"),e.prototype.init=function(e){var t=e.networkStatus||m.loading;return this.variables&&this.networkStatus!==m.loading&&!oI(this.variables,e.variables)&&(t=m.setVariables),oI(e.variables,this.variables)||(this.lastDiff=void 0),Object.assign(this,{document:e.document,variables:e.variables,networkError:null,graphQLErrors:this.graphQLErrors||[],networkStatus:t}),e.observableQuery&&this.setObservableQuery(e.observableQuery),e.lastRequestId&&(this.lastRequestId=e.lastRequestId),this},e.prototype.reset=function(){iV(this),this.lastDiff=void 0,this.dirty=!1},e.prototype.getDiff=function(e){void 0===e&&(e=this.variables);var t=this.getDiffOptions(e);if(this.lastDiff&&oI(t,this.lastDiff.options))return this.lastDiff.diff;this.updateWatch(this.variables=e);var n=this.observableQuery;if(n&&"no-cache"===n.options.fetchPolicy)return{complete:!1};var o=this.cache.diff(t);return this.updateLastDiff(o,t),o},e.prototype.updateLastDiff=function(e,t){this.lastDiff=e?{diff:e,options:t||this.getDiffOptions()}:void 0},e.prototype.getDiffOptions=function(e){var t;return void 0===e&&(e=this.variables),{query:this.document,variables:e,returnPartialData:!0,optimistic:!0,canonizeResults:null===(t=this.observableQuery)||void 0===t?void 0:t.options.canonizeResults}},e.prototype.setDiff=function(e){var t=this,n=this.lastDiff&&this.lastDiff.diff;this.updateLastDiff(e),this.dirty||oI(n&&n.result,e&&e.result)||(this.dirty=!0,this.notifyTimeout||(this.notifyTimeout=setTimeout(function(){return t.notify()},0)))},e.prototype.setObservableQuery=function(e){var t=this;e!==this.observableQuery&&(this.oqListener&&this.listeners.delete(this.oqListener),this.observableQuery=e,e?(e.queryInfo=this,this.listeners.add(this.oqListener=function(){t.getDiff().fromOptimisticTransaction?e.observe():iI(e)})):delete this.oqListener)},e.prototype.notify=function(){var e=this;iV(this),this.shouldNotify()&&this.listeners.forEach(function(t){return t(e)}),this.dirty=!1},e.prototype.shouldNotify=function(){if(!this.dirty||!this.listeners.size)return!1;if(iD(this.networkStatus)&&this.observableQuery){var e=this.observableQuery.options.fetchPolicy;if("cache-only"!==e&&"cache-and-network"!==e)return!1}return!0},e.prototype.stop=function(){if(!this.stopped){this.stopped=!0,this.reset(),this.cancel(),this.cancel=e.prototype.cancel,this.subscriptions.forEach(function(e){return e.unsubscribe()});var t=this.observableQuery;t&&t.stopPolling()}},e.prototype.cancel=function(){},e.prototype.updateWatch=function(e){var t=this;void 0===e&&(e=this.variables);var n=this.observableQuery;if(!n||"no-cache"!==n.options.fetchPolicy){var o=eT(eT({},this.getDiffOptions(e)),{watcher:this,callback:function(e){return t.setDiff(e)}});this.lastWatch&&oI(o,this.lastWatch)||(this.cancel(),this.cancel=this.cache.watch(this.lastWatch=o))}},e.prototype.resetLastWrite=function(){this.lastWrite=void 0},e.prototype.shouldWrite=function(e,t){var n=this.lastWrite;return!(n&&n.dmCount===iL.get(this.cache)&&oI(t,n.variables)&&oI(e.data,n.result.data))},e.prototype.markResult=function(e,t,n){var o=this;this.graphQLErrors=n9(e.errors)?e.errors:[],this.reset(),"no-cache"===t.fetchPolicy?this.updateLastDiff({result:e.data,complete:!0},this.getDiffOptions(t.variables)):0!==n&&(iZ(e,t.errorPolicy)?this.cache.performTransaction(function(r){if(o.shouldWrite(e,t.variables))r.writeQuery({query:o.document,data:e.data,variables:t.variables,overwrite:1===n}),o.lastWrite={result:e,variables:t.variables,dmCount:iL.get(o.cache)};else if(o.lastDiff&&o.lastDiff.diff.complete){e.data=o.lastDiff.diff.result;return}var i=o.getDiffOptions(t.variables),a=r.diff(i);o.stopped||o.updateWatch(t.variables),o.updateLastDiff(a,i),a.complete&&(e.data=a.result)}):this.lastWrite=void 0)},e.prototype.markReady=function(){return this.networkError=null,this.networkStatus=m.ready},e.prototype.markError=function(e){return this.networkStatus=m.error,this.lastWrite=void 0,this.reset(),e.graphQLErrors&&(this.graphQLErrors=e.graphQLErrors),e.networkError&&(this.networkError=e.networkError),e},e}();function iZ(e,t){void 0===t&&(t="none");var n="ignore"===t||"all"===t,o=!n7(e);return!o&&n&&e.data&&(o=!0),o}ew(iZ,"shouldWriteResult");var iB=Object.prototype.hasOwnProperty,iQ=function(){function e(e){var t=e.cache,n=e.link,o=e.defaultOptions,r=e.queryDeduplication,i=e.onBroadcast,a=e.ssrMode,s=e.clientAwareness,l=e.localState,c=e.assumeImmutableResults;this.clientAwareness={},this.queries=new Map,this.fetchCancelFns=new Map,this.transformCache=new(nJ?WeakMap:Map),this.queryIdCounter=1,this.requestIdCounter=1,this.mutationIdCounter=1,this.inFlightLinkObservables=new Map,this.cache=t,this.link=n,this.defaultOptions=o||Object.create(null),this.queryDeduplication=void 0!==r&&r,this.clientAwareness=void 0===s?{}:s,this.localState=l||new iR({cache:t}),this.ssrMode=void 0!==a&&a,this.assumeImmutableResults=!!c,(this.onBroadcast=i)&&(this.mutationStore=Object.create(null))}return ew(e,"QueryManager"),e.prototype.stop=function(){var e=this;this.queries.forEach(function(t,n){e.stopQueryNoBroadcast(n)}),this.cancelPendingFetches(__DEV__?new eA("QueryManager stopped while query was in flight"):new eA(11))},e.prototype.cancelPendingFetches=function(e){this.fetchCancelFns.forEach(function(t){return t(e)}),this.fetchCancelFns.clear()},e.prototype.mutate=function(e){var t,n,o=e.mutation,r=e.variables,i=e.optimisticResponse,a=e.updateQueries,s=e.refetchQueries,l=void 0===s?[]:s,c=e.awaitRefetchQueries,u=void 0!==c&&c,d=e.update,h=e.onQueryUpdated,p=e.fetchPolicy,f=void 0===p?(null===(t=this.defaultOptions.mutate)||void 0===t?void 0:t.fetchPolicy)||"network-only":p,m=e.errorPolicy,g=void 0===m?(null===(n=this.defaultOptions.mutate)||void 0===n?void 0:n.errorPolicy)||"none":m,v=e.keepRootFields,b=e.context;return eM(this,void 0,void 0,function(){var e,t,n;return eI(this,function(s){switch(s.label){case 0:if(__DEV__?eV(o,"mutation option is required. You must specify your GraphQL document in the mutation option."):eV(o,12),__DEV__?eV("network-only"===f||"no-cache"===f,"Mutations support only 'network-only' or 'no-cache' fetchPolicy strings. The default `network-only` behavior automatically writes mutation results to the cache. Passing `no-cache` skips the cache write."):eV("network-only"===f||"no-cache"===f,13),e=this.generateMutationId(),o=this.transform(o).document,r=this.getVariables(o,r),!this.transform(o).hasClientExports)return[3,2];return[4,this.localState.addExportedVariables(o,r,b)];case 1:r=s.sent(),s.label=2;case 2:return t=this.mutationStore&&(this.mutationStore[e]={mutation:o,variables:r,loading:!0,error:null}),i&&this.markMutationOptimistic(i,{mutationId:e,document:o,variables:r,fetchPolicy:f,errorPolicy:g,context:b,updateQueries:a,update:d,keepRootFields:v}),this.broadcastQueries(),n=this,[2,new Promise(function(s,c){return nX(n.getObservableFromLink(o,eT(eT({},b),{optimisticResponse:i}),r,!1),function(s){if(n7(s)&&"none"===g)throw new iO({graphQLErrors:s.errors});t&&(t.loading=!1,t.error=null);var c=eT({},s);return"function"==typeof l&&(l=l(c)),"ignore"===g&&n7(c)&&delete c.errors,n.markMutationResult({mutationId:e,result:c,document:o,variables:r,fetchPolicy:f,errorPolicy:g,context:b,update:d,updateQueries:a,awaitRefetchQueries:u,refetchQueries:l,removeOptimistic:i?e:void 0,onQueryUpdated:h,keepRootFields:v})}).subscribe({next:function(e){n.broadcastQueries(),s(e)},error:function(o){t&&(t.loading=!1,t.error=o),i&&n.cache.removeOptimistic(e),n.broadcastQueries(),c(o instanceof iO?o:new iO({networkError:o}))}})})]}})})},e.prototype.markMutationResult=function(e,t){var n=this;void 0===t&&(t=this.cache);var o=e.result,r=[],i="no-cache"===e.fetchPolicy;if(!i&&iZ(o,e.errorPolicy)){r.push({result:o.data,dataId:"ROOT_MUTATION",query:e.document,variables:e.variables});var a=e.updateQueries;a&&this.queries.forEach(function(e,i){var s=e.observableQuery,l=s&&s.queryName;if(l&&iB.call(a,l)){var c=a[l],u=n.queries.get(i),d=u.document,h=u.variables,p=t.diff({query:d,variables:h,returnPartialData:!0,optimistic:!1}),f=p.result;if(p.complete&&f){var m=c(f,{mutationResult:o,queryName:d&&t2(d)||void 0,queryVariables:h});m&&r.push({result:m,dataId:"ROOT_QUERY",query:d,variables:h})}}})}if(r.length>0||e.refetchQueries||e.update||e.onQueryUpdated||e.removeOptimistic){var s=[];if(this.refetchQueries({updateCache:function(t){i||r.forEach(function(e){return t.write(e)});var a=e.update;if(a){if(!i){var s=t.diff({id:"ROOT_MUTATION",query:n.transform(e.document).asQuery,variables:e.variables,optimistic:!1,returnPartialData:!0});s.complete&&(o=eT(eT({},o),{data:s.result}))}a(t,o,{context:e.context,variables:e.variables})}i||e.keepRootFields||t.modify({id:"ROOT_MUTATION",fields:function(e,t){var n=t.fieldName,o=t.DELETE;return"__typename"===n?e:o}})},include:e.refetchQueries,optimistic:!1,removeOptimistic:e.removeOptimistic,onQueryUpdated:e.onQueryUpdated||null}).forEach(function(e){return s.push(e)}),e.awaitRefetchQueries||e.onQueryUpdated)return Promise.all(s).then(function(){return o})}return Promise.resolve(o)},e.prototype.markMutationOptimistic=function(e,t){var n=this,o="function"==typeof e?e(t.variables):e;return this.cache.recordOptimisticTransaction(function(e){try{n.markMutationResult(eT(eT({},t),{result:{data:o}}),e)}catch(e){__DEV__&&eV.error(e)}},t.mutationId)},e.prototype.fetchQuery=function(e,t,n){return this.fetchQueryObservable(e,t,n).promise},e.prototype.getQueryStore=function(){var e=Object.create(null);return this.queries.forEach(function(t,n){e[n]={variables:t.variables,networkStatus:t.networkStatus,networkError:t.networkError,graphQLErrors:t.graphQLErrors}}),e},e.prototype.resetErrors=function(e){var t=this.queries.get(e);t&&(t.networkError=void 0,t.graphQLErrors=[])},e.prototype.transform=function(e){var t=this.transformCache;if(!t.has(e)){var n=this.cache.transformDocument(e),o=na(this.cache.transformForLink(n)),r=this.localState.clientQuery(n),i=o&&this.localState.serverQuery(o),a={document:n,hasClientExports:tC(n),hasForcedResolvers:this.localState.shouldForceResolvers(n),clientQuery:r,serverQuery:i,defaultVars:t8(t1(n)),asQuery:eT(eT({},n),{definitions:n.definitions.map(function(e){return"OperationDefinition"===e.kind&&"query"!==e.operation?eT(eT({},e),{operation:"query"}):e})})},s=ew(function(e){e&&!t.has(e)&&t.set(e,a)},"add");s(e),s(n),s(r),s(i)}return t.get(e)},e.prototype.getVariables=function(e,t){return eT(eT({},this.transform(e).defaultVars),t)},e.prototype.watchQuery=function(e){void 0===(e=eT(eT({},e),{variables:this.getVariables(e.query,e.variables)})).notifyOnNetworkStatusChange&&(e.notifyOnNetworkStatusChange=!1);var t=new iq(this),n=new iM({queryManager:this,queryInfo:t,options:e});return this.queries.set(n.queryId,t),t.init({document:n.query,observableQuery:n,variables:n.variables}),n},e.prototype.query=function(e,t){var n=this;return void 0===t&&(t=this.generateQueryId()),__DEV__?eV(e.query,"query option is required. You must specify your GraphQL document in the query option."):eV(e.query,14),__DEV__?eV("Document"===e.query.kind,'You must wrap the query string in a "gql" tag.'):eV("Document"===e.query.kind,15),__DEV__?eV(!e.returnPartialData,"returnPartialData option only supported on watchQuery."):eV(!e.returnPartialData,16),__DEV__?eV(!e.pollInterval,"pollInterval option only supported on watchQuery."):eV(!e.pollInterval,17),this.fetchQuery(t,e).finally(function(){return n.stopQuery(t)})},e.prototype.generateQueryId=function(){return String(this.queryIdCounter++)},e.prototype.generateRequestId=function(){return this.requestIdCounter++},e.prototype.generateMutationId=function(){return String(this.mutationIdCounter++)},e.prototype.stopQueryInStore=function(e){this.stopQueryInStoreNoBroadcast(e),this.broadcastQueries()},e.prototype.stopQueryInStoreNoBroadcast=function(e){var t=this.queries.get(e);t&&t.stop()},e.prototype.clearStore=function(e){return void 0===e&&(e={discardWatches:!0}),this.cancelPendingFetches(__DEV__?new eA("Store reset while query was in flight (not completed in link chain)"):new eA(18)),this.queries.forEach(function(e){e.observableQuery?e.networkStatus=m.loading:e.stop()}),this.mutationStore&&(this.mutationStore=Object.create(null)),this.cache.reset(e)},e.prototype.getObservableQueries=function(e){var t=this;void 0===e&&(e="active");var n=new Map,o=new Map,r=new Set;return Array.isArray(e)&&e.forEach(function(e){"string"==typeof e?o.set(e,!1):tz(e)?o.set(t.transform(e).document,!1):tO(e)&&e.query&&r.add(e)}),this.queries.forEach(function(t,r){var i=t.observableQuery,a=t.document;if(i){if("all"===e){n.set(r,i);return}var s=i.queryName;if("standby"===i.options.fetchPolicy||"active"===e&&!i.hasObservers())return;("active"===e||s&&o.has(s)||a&&o.has(a))&&(n.set(r,i),s&&o.set(s,!0),a&&o.set(a,!0))}}),r.size&&r.forEach(function(e){var o=on("legacyOneTimeQuery"),r=t.getQuery(o).init({document:e.query,variables:e.variables}),i=new iM({queryManager:t,queryInfo:r,options:eT(eT({},e),{fetchPolicy:"network-only"})});eV(i.queryId===o),r.setObservableQuery(i),n.set(o,i)}),__DEV__&&o.size&&o.forEach(function(e,t){!e&&__DEV__&&eV.warn("Unknown query ".concat("string"==typeof t?"named ":"").concat(JSON.stringify(t,null,2)," requested in refetchQueries options.include array"))}),n},e.prototype.reFetchObservableQueries=function(e){var t=this;void 0===e&&(e=!1);var n=[];return this.getObservableQueries(e?"all":"active").forEach(function(o,r){var i=o.options.fetchPolicy;o.resetLastResults(),(e||"standby"!==i&&"cache-only"!==i)&&n.push(o.refetch()),t.getQuery(r).setDiff(null)}),this.broadcastQueries(),Promise.all(n)},e.prototype.setObservableQuery=function(e){this.getQuery(e.queryId).setObservableQuery(e)},e.prototype.startGraphQLSubscription=function(e){var t=this,n=e.query,o=e.fetchPolicy,r=e.errorPolicy,i=e.variables,a=e.context,s=void 0===a?{}:a;n=this.transform(n).document,i=this.getVariables(n,i);var l=ew(function(e){return t.getObservableFromLink(n,s,e).map(function(i){if("no-cache"!==o&&(iZ(i,r)&&t.cache.write({query:n,result:i.data,dataId:"ROOT_SUBSCRIPTION",variables:e}),t.broadcastQueries()),n7(i))throw new iO({graphQLErrors:i.errors});return i})},"makeObservable");if(this.transform(n).hasClientExports){var c=this.localState.addExportedVariables(n,i,s).then(l);return new nV(function(e){var t=null;return c.then(function(n){return t=n.subscribe(e)},e.error),function(){return t&&t.unsubscribe()}})}return l(i)},e.prototype.stopQuery=function(e){this.stopQueryNoBroadcast(e),this.broadcastQueries()},e.prototype.stopQueryNoBroadcast=function(e){this.stopQueryInStoreNoBroadcast(e),this.removeQuery(e)},e.prototype.removeQuery=function(e){this.fetchCancelFns.delete(e),this.queries.has(e)&&(this.getQuery(e).stop(),this.queries.delete(e))},e.prototype.broadcastQueries=function(){this.onBroadcast&&this.onBroadcast(),this.queries.forEach(function(e){return e.notify()})},e.prototype.getLocalState=function(){return this.localState},e.prototype.getObservableFromLink=function(e,t,n,o){var r,i,a=this;void 0===o&&(o=null!==(r=null==t?void 0:t.queryDeduplication)&&void 0!==r?r:this.queryDeduplication);var s=this.transform(e).serverQuery;if(s){var l=this.inFlightLinkObservables,c=this.link,u={query:s,variables:n,operationName:t2(s)||void 0,context:this.prepareContext(eT(eT({},t),{forceFetch:!o}))};if(t=u.context,o){var d=l.get(s)||new Map;l.set(s,d);var h=rW(n);if(!(i=d.get(h))){var p=new n8([om(c,u)]);d.set(h,i=p),p.cleanup(function(){d.delete(h)&&d.size<1&&l.delete(s)})}}else i=new n8([om(c,u)])}else i=new n8([nV.of({data:{}})]),t=this.prepareContext(t);var f=this.transform(e).clientQuery;return f&&(i=nX(i,function(e){return a.localState.runResolvers({document:f,remoteResult:e,context:t,variables:n})})),i},e.prototype.getResultsFromLink=function(e,t,n){var o=e.lastRequestId=this.generateRequestId();return nX(this.getObservableFromLink(e.document,n.context,n.variables),function(r){var i=n9(r.errors);if(o>=e.lastRequestId){if(i&&"none"===n.errorPolicy)throw e.markError(new iO({graphQLErrors:r.errors}));e.markResult(r,n,t),e.markReady()}var a={data:r.data,loading:!1,networkStatus:m.ready};return i&&"ignore"!==n.errorPolicy&&(a.errors=r.errors,a.networkStatus=m.error),a},function(t){var n=i$(t)?t:new iO({networkError:t});throw o>=e.lastRequestId&&e.markError(n),n})},e.prototype.fetchQueryObservable=function(e,t,n){var o=this;void 0===n&&(n=m.loading);var r=this.transform(t.query).document,i=this.getVariables(r,t.variables),a=this.getQuery(e),s=this.defaultOptions.watchQuery,l=t.fetchPolicy,c=void 0===l?s&&s.fetchPolicy||"cache-first":l,u=t.errorPolicy,d=void 0===u?s&&s.errorPolicy||"none":u,h=t.returnPartialData,p=t.notifyOnNetworkStatusChange,f=t.context,g=Object.assign({},t,{query:r,variables:i,fetchPolicy:c,errorPolicy:d,returnPartialData:void 0!==h&&h,notifyOnNetworkStatusChange:void 0!==p&&p,context:void 0===f?{}:f}),v=ew(function(e){return g.variables=e,o.fetchQueryByPolicy(a,g,n)},"fromVariables");this.fetchCancelFns.set(e,function(e){setTimeout(function(){return b.cancel(e)})});var b=new n8(this.transform(g.query).hasClientExports?this.localState.addExportedVariables(g.query,g.variables,g.context).then(v):v(g.variables));return b.cleanup(function(){o.fetchCancelFns.delete(e),a.observableQuery&&a.observableQuery.applyNextFetchPolicy("after-fetch",t)}),b},e.prototype.refetchQueries=function(e){var t=this,n=e.updateCache,o=e.include,r=e.optimistic,i=void 0!==r&&r,a=e.removeOptimistic,s=void 0===a?i?on("refetchQueries"):void 0:a,l=e.onQueryUpdated,c=new Map;o&&this.getObservableQueries(o).forEach(function(e,n){c.set(n,{oq:e,lastDiff:t.getQuery(n).getDiff()})});var u=new Map;return n&&this.cache.batch({update:n,optimistic:i&&s||!1,removeOptimistic:s,onWatchUpdated:function(e,t,n){var o=e.watcher instanceof iq&&e.watcher.observableQuery;if(o){if(l){c.delete(o.queryId);var r=l(o,t,n);return!0===r&&(r=o.refetch()),!1!==r&&u.set(o,r),r}null!==l&&c.set(o.queryId,{oq:o,lastDiff:n,diff:t})}}}),c.size&&c.forEach(function(e,n){var o,r=e.oq,i=e.lastDiff,a=e.diff;if(l){if(!a){var s=r.queryInfo;s.reset(),a=s.getDiff()}o=l(r,a,i)}l&&!0!==o||(o=r.refetch()),!1!==o&&u.set(r,o),n.indexOf("legacyOneTimeQuery")>=0&&t.stopQueryNoBroadcast(n)}),s&&this.cache.removeOptimistic(s),u},e.prototype.fetchQueryByPolicy=function(e,t,n){var o=this,r=t.query,i=t.variables,a=t.fetchPolicy,s=t.refetchWritePolicy,l=t.errorPolicy,c=t.returnPartialData,u=t.context,d=t.notifyOnNetworkStatusChange,h=e.networkStatus;e.init({document:this.transform(r).document,variables:i,networkStatus:n});var p=ew(function(){return e.getDiff(i)},"readCache"),f=ew(function(t,n){void 0===n&&(n=e.networkStatus||m.loading);var a=t.result;!__DEV__||c||oI(a,{})||ij(t.missing);var s=ew(function(e){return nV.of(eT({data:e,loading:iD(n),networkStatus:n},t.complete?null:{partial:!0}))},"fromData");return a&&o.transform(r).hasForcedResolvers?o.localState.runResolvers({document:r,remoteResult:{data:a},context:u,variables:i,onlyRunForcedResolvers:!0}).then(function(e){return s(e.data||void 0)}):s(a)},"resultsFromCache"),g="no-cache"===a?0:n===m.refetch&&"merge"!==s?1:2,v=ew(function(){return o.getResultsFromLink(e,g,{variables:i,context:u,fetchPolicy:a,errorPolicy:l})},"resultsFromLink"),b=d&&"number"==typeof h&&h!==n&&iD(n);switch(a){default:case"cache-first":var y=p();if(y.complete)return[f(y,e.markReady())];if(c||b)return[f(y),v()];return[v()];case"cache-and-network":var y=p();if(y.complete||c||b)return[f(y),v()];return[v()];case"cache-only":return[f(p(),e.markReady())];case"network-only":if(b)return[f(p()),v()];return[v()];case"no-cache":if(b)return[f(e.getDiff()),v()];return[v()];case"standby":return[]}},e.prototype.getQuery=function(e){return e&&!this.queries.has(e)&&this.queries.set(e,new iq(this,e)),this.queries.get(e)},e.prototype.prepareContext=function(e){void 0===e&&(e={});var t=this.localState.prepareContext(e);return eT(eT({},t),{clientAwareness:this.clientAwareness})},e}(),iH=!1,iG=function(){function e(e){var t=this;this.resetStoreCallbacks=[],this.clearStoreCallbacks=[];var n=e.uri,o=e.credentials,r=e.headers,i=e.cache,a=e.ssrMode,s=void 0!==a&&a,l=e.ssrForceFetchDelay,c=void 0===l?0:l,u=e.connectToDevTools,d=void 0===u?"object"==typeof window&&!window.__APOLLO_CLIENT__&&__DEV__:u,h=e.queryDeduplication,p=void 0===h||h,f=e.defaultOptions,m=e.assumeImmutableResults,g=e.resolvers,v=e.typeDefs,b=e.fragmentMatcher,y=e.name,k=e.version,w=e.link;if(w||(w=n?new oE({uri:n,credentials:o,headers:r}):of.empty()),!i)throw __DEV__?new eA("To initialize Apollo Client, you must specify a 'cache' property in the options object. \nFor more information, please visit: https://go.apollo.dev/c/docs"):new eA(7);if(this.link=w,this.cache=i,this.disableNetworkFetches=s||c>0,this.queryDeduplication=p,this.defaultOptions=f||Object.create(null),this.typeDefs=v,c&&setTimeout(function(){return t.disableNetworkFetches=!1},c),this.watchQuery=this.watchQuery.bind(this),this.query=this.query.bind(this),this.mutate=this.mutate.bind(this),this.resetStore=this.resetStore.bind(this),this.reFetchObservableQueries=this.reFetchObservableQueries.bind(this),d&&"object"==typeof window&&(window.__APOLLO_CLIENT__=this),!iH&&__DEV__&&(iH=!0,"undefined"!=typeof window&&window.document&&window.top===window.self&&!window.__APOLLO_DEVTOOLS_GLOBAL_HOOK__)){var x=window.navigator,S=x&&x.userAgent,C=void 0;"string"==typeof S&&(S.indexOf("Chrome/")>-1?C="https://chrome.google.com/webstore/detail/apollo-client-developer-t/jdkknkkbebbapilgoeccciglkfbmbnfm":S.indexOf("Firefox/")>-1&&(C="https://addons.mozilla.org/en-US/firefox/addon/apollo-developer-tools/")),C&&__DEV__&&eV.log("Download the Apollo DevTools for a better development experience: "+C)}this.version="3.6.5",this.localState=new iR({cache:i,client:this,resolvers:g,fragmentMatcher:b}),this.queryManager=new iQ({cache:this.cache,link:this.link,defaultOptions:this.defaultOptions,queryDeduplication:p,ssrMode:s,clientAwareness:{name:y,version:k},localState:this.localState,assumeImmutableResults:void 0!==m&&m,onBroadcast:d?function(){t.devToolsHookCb&&t.devToolsHookCb({action:{},state:{queries:t.queryManager.getQueryStore(),mutations:t.queryManager.mutationStore||{}},dataWithOptimisticResults:t.cache.extract(!0)})}:void 0})}return ew(e,"ApolloClient"),e.prototype.stop=function(){this.queryManager.stop()},e.prototype.watchQuery=function(e){return this.defaultOptions.watchQuery&&(e=or(this.defaultOptions.watchQuery,e)),this.disableNetworkFetches&&("network-only"===e.fetchPolicy||"cache-and-network"===e.fetchPolicy)&&(e=eT(eT({},e),{fetchPolicy:"cache-first"})),this.queryManager.watchQuery(e)},e.prototype.query=function(e){return this.defaultOptions.query&&(e=or(this.defaultOptions.query,e)),__DEV__?eV("cache-and-network"!==e.fetchPolicy,"The cache-and-network fetchPolicy does not work with client.query, because client.query can only return a single result. Please use client.watchQuery to receive multiple results from the cache and the network, or consider using a different fetchPolicy, such as cache-first or network-only."):eV("cache-and-network"!==e.fetchPolicy,8),this.disableNetworkFetches&&"network-only"===e.fetchPolicy&&(e=eT(eT({},e),{fetchPolicy:"cache-first"})),this.queryManager.query(e)},e.prototype.mutate=function(e){return this.defaultOptions.mutate&&(e=or(this.defaultOptions.mutate,e)),this.queryManager.mutate(e)},e.prototype.subscribe=function(e){return this.queryManager.startGraphQLSubscription(e)},e.prototype.readQuery=function(e,t){return void 0===t&&(t=!1),this.cache.readQuery(e,t)},e.prototype.readFragment=function(e,t){return void 0===t&&(t=!1),this.cache.readFragment(e,t)},e.prototype.writeQuery=function(e){this.cache.writeQuery(e),this.queryManager.broadcastQueries()},e.prototype.writeFragment=function(e){this.cache.writeFragment(e),this.queryManager.broadcastQueries()},e.prototype.__actionHookForDevTools=function(e){this.devToolsHookCb=e},e.prototype.__requestRaw=function(e){return om(this.link,e)},e.prototype.resetStore=function(){var e=this;return Promise.resolve().then(function(){return e.queryManager.clearStore({discardWatches:!1})}).then(function(){return Promise.all(e.resetStoreCallbacks.map(function(e){return e()}))}).then(function(){return e.reFetchObservableQueries()})},e.prototype.clearStore=function(){var e=this;return Promise.resolve().then(function(){return e.queryManager.clearStore({discardWatches:!0})}).then(function(){return Promise.all(e.clearStoreCallbacks.map(function(e){return e()}))})},e.prototype.onResetStore=function(e){var t=this;return this.resetStoreCallbacks.push(e),function(){t.resetStoreCallbacks=t.resetStoreCallbacks.filter(function(t){return t!==e})}},e.prototype.onClearStore=function(e){var t=this;return this.clearStoreCallbacks.push(e),function(){t.clearStoreCallbacks=t.clearStoreCallbacks.filter(function(t){return t!==e})}},e.prototype.reFetchObservableQueries=function(e){return this.queryManager.reFetchObservableQueries(e)},e.prototype.refetchQueries=function(e){var t=this.queryManager.refetchQueries(e),n=[],o=[];t.forEach(function(e,t){n.push(t),o.push(e)});var r=Promise.all(o);return r.queries=n,r.results=o,r.catch(function(e){__DEV__&&eV.debug("In client.refetchQueries, Promise.all promise rejected with error ".concat(e))}),r},e.prototype.getObservableQueries=function(e){return void 0===e&&(e="active"),this.queryManager.getObservableQueries(e)},e.prototype.extract=function(e){return this.cache.extract(e)},e.prototype.restore=function(e){return this.cache.restore(e)},e.prototype.addResolvers=function(e){this.localState.addResolvers(e)},e.prototype.setResolvers=function(e){this.localState.setResolvers(e)},e.prototype.getResolvers=function(){return this.localState.getResolvers()},e.prototype.setLocalStateFragmentMatcher=function(e){this.localState.setFragmentMatcher(e)},e.prototype.setLink=function(e){this.link=this.queryManager.link=e},e}();eQ(eK?"log":"silent");var iU=n1?Symbol.for("__APOLLO_CONTEXT__"):"__APOLLO_CONTEXT__";function iW(){var e=b.createContext[iU];return e||(Object.defineProperty(b.createContext,iU,{value:e=b.createContext({}),enumerable:!1,writable:!1,configurable:!0}),e.displayName="ApolloContext"),e}ew(iW,"getApolloContext");var iK=ew(function(e){var t=e.client,n=e.children,o=iW();return b.createElement(o.Consumer,null,function(e){return void 0===e&&(e={}),t&&e.client!==t&&(e=Object.assign({},e,{client:t})),__DEV__?eV(e.client,'ApolloProvider was not passed a client instance. Make sure you pass in your client via the "client" prop.'):eV(e.client,26),b.createElement(o.Provider,{value:e},n)})},"ApolloProvider");function iY(e){var t=(0,b.useContext)(iW()),n=e||t.client;return __DEV__?eV(!!n,'Could not find "client" in the context or passed in as an option. Wrap the root component in an <ApolloProvider>, or pass an ApolloClient instance in via options.'):eV(!!n,29),n}ew(iY,"useApolloClient");var iX=!1,iJ=(u||(u=n.t(b,2))).useSyncExternalStore||function(e,t,n){var o=t();__DEV__&&!iX&&o!==t()&&(iX=!0,__DEV__&&eV.error("The result of getSnapshot should be cached to avoid an infinite loop"));var r=b.useState({inst:{value:o,getSnapshot:t}}),i=r[0].inst,a=r[1];return n5?b.useLayoutEffect(function(){Object.assign(i,{value:o,getSnapshot:t}),i0(i)&&a({inst:i})},[e,o,t]):Object.assign(i,{value:o,getSnapshot:t}),b.useEffect(function(){return i0(i)&&a({inst:i}),e(ew(function(){i0(i)&&a({inst:i})},"handleStoreChange"))},[e]),o};function i0(e){var t=e.value,n=e.getSnapshot;try{return t!==n()}catch(e){return!0}}ew(i0,"checkIfSnapshotChanged"),(s=g||(g={}))[s.Query=0]="Query",s[s.Mutation=1]="Mutation",s[s.Subscription=2]="Subscription";var i1=new Map;function i2(e){var t;switch(e){case g.Query:t="Query";break;case g.Mutation:t="Mutation";break;case g.Subscription:t="Subscription"}return t}function i3(e){var t,n,o=i1.get(e);if(o)return o;__DEV__?eV(!!e&&!!e.kind,"Argument of ".concat(e," passed to parser was not a valid GraphQL ")+"DocumentNode. You may need to use 'graphql-tag' or another method to convert your operation into a document"):eV(!!e&&!!e.kind,30);for(var r=[],i=[],a=[],s=[],l=0,c=e.definitions;l<c.length;l++){var u=c[l];if("FragmentDefinition"===u.kind){r.push(u);continue}if("OperationDefinition"===u.kind)switch(u.operation){case"query":i.push(u);break;case"mutation":a.push(u);break;case"subscription":s.push(u)}}__DEV__?eV(!r.length||i.length||a.length||s.length,"Passing only a fragment to 'graphql' is not yet supported. You must include a query, subscription or mutation as well"):eV(!r.length||i.length||a.length||s.length,31),__DEV__?eV(i.length+a.length+s.length<=1,"react-apollo only supports a query, subscription, or a mutation per HOC. "+"".concat(e," had ").concat(i.length," queries, ").concat(s.length," ")+"subscriptions and ".concat(a.length," mutations. ")+"You can use 'compose' to join multiple operation types to a component"):eV(i.length+a.length+s.length<=1,32),n=i.length?g.Query:g.Mutation,i.length||a.length||(n=g.Subscription);var d=i.length?i:a.length?a:s;__DEV__?eV(1===d.length,"react-apollo only supports one definition per HOC. ".concat(e," had ")+"".concat(d.length," definitions. ")+"You can use 'compose' to join multiple operation types to a component"):eV(1===d.length,33);var h=d[0];t=h.variableDefinitions||[];var p={name:h.name&&"Name"===h.name.kind?h.name.value:"data",type:n,variables:t};return i1.set(e,p),p}function i5(e,t){var n=i3(e),o=i2(t),r=i2(n.type);__DEV__?eV(n.type===t,"Running a ".concat(o," requires a graphql ")+"".concat(o,", but a ").concat(r," was used instead.")):eV(n.type===t,34)}ew(i2,"operationName"),ew(i3,"parser"),ew(i5,"verifyDocumentType");var i4=Object.prototype.hasOwnProperty;function i6(e,t){return void 0===t&&(t=Object.create(null)),i8(iY(t.client),e).useQuery(t)}function i8(e,t){var n=(0,b.useRef)();n.current&&e===n.current.client&&t===n.current.query||(n.current=new i9(e,t,n.current));var o=n.current,r=(0,b.useState)(0),i=(r[0],r[1]);return o.forceUpdate=function(){i(function(e){return e+1})},o}ew(i6,"useQuery"),ew(i8,"useInternalState");var i9=function(){function e(e,t,n){this.client=e,this.query=t,this.asyncResolveFns=new Set,this.optionsToIgnoreOnce=new(n0?WeakSet:Set),this.ssrDisabledResult=nK({loading:!0,data:void 0,error:void 0,networkStatus:m.loading}),this.skipStandbyResult=nK({loading:!1,data:void 0,error:void 0,networkStatus:m.ready}),this.toQueryResultCache=new(nJ?WeakMap:Map),i5(t,g.Query);var o=n&&n.result,r=o&&o.data;r&&(this.previousData=r)}return ew(e,"InternalState"),e.prototype.forceUpdate=function(){__DEV__&&eV.warn("Calling default no-op implementation of InternalState#forceUpdate")},e.prototype.asyncUpdate=function(){var e=this;return new Promise(function(t){e.asyncResolveFns.add(t),e.optionsToIgnoreOnce.add(e.watchQueryOptions),e.forceUpdate()})},e.prototype.useQuery=function(e){var t=this;this.renderPromises=(0,b.useContext)(iW()).renderPromises,this.useOptions(e);var n=this.useObservableQuery(),o=iJ((0,b.useCallback)(function(){if(t.renderPromises)return function(){};var e=ew(function(){var e=t.result,o=n.getCurrentResult();e&&e.loading===o.loading&&e.networkStatus===o.networkStatus&&oI(e.data,o.data)||t.setResult(o)},"onNext"),o=ew(function(i){var a=n.last;r.unsubscribe();try{n.resetLastResults(),r=n.subscribe(e,o)}finally{n.last=a}if(!i4.call(i,"graphQLErrors"))throw i;var s=t.result;(!s||s&&s.loading||!oI(i,s.error))&&t.setResult({data:s&&s.data,error:i,loading:!1,networkStatus:m.error})},"onError"),r=n.subscribe(e,o);return function(){return r.unsubscribe()}},[n,this.renderPromises,this.client.disableNetworkFetches]),function(){return t.getCurrentResult()},function(){return t.getCurrentResult()});this.unsafeHandlePartialRefetch(o);var r=this.toQueryResult(o);return!r.loading&&this.asyncResolveFns.size&&(this.asyncResolveFns.forEach(function(e){return e(r)}),this.asyncResolveFns.clear()),r},e.prototype.useOptions=function(t){var n,o=this.createWatchQueryOptions(this.queryHookOptions=t),r=this.watchQueryOptions;(this.optionsToIgnoreOnce.has(r)||!oI(o,r))&&(this.watchQueryOptions=o,r&&this.observable&&(this.optionsToIgnoreOnce.delete(r),this.observable.reobserve(o),this.previousData=(null===(n=this.result)||void 0===n?void 0:n.data)||this.previousData,this.result=void 0)),this.onCompleted=t.onCompleted||e.prototype.onCompleted,this.onError=t.onError||e.prototype.onError,(this.renderPromises||this.client.disableNetworkFetches)&&!1===this.queryHookOptions.ssr&&!this.queryHookOptions.skip?this.result=this.ssrDisabledResult:this.queryHookOptions.skip||"standby"===this.watchQueryOptions.fetchPolicy?this.result=this.skipStandbyResult:(this.result===this.ssrDisabledResult||this.result===this.skipStandbyResult)&&(this.result=void 0)},e.prototype.createWatchQueryOptions=function(e){void 0===e&&(e={});var t,n=e.skip,o=Object.assign((e.ssr,e.onCompleted,e.onError,e.displayName,e.defaultOptions,ez(e,["skip","ssr","onCompleted","onError","displayName","defaultOptions"])),{query:this.query});if(this.renderPromises&&("network-only"===o.fetchPolicy||"cache-and-network"===o.fetchPolicy)&&(o.fetchPolicy="cache-first"),o.variables||(o.variables={}),n){var r=o.fetchPolicy,i=void 0===r?this.getDefaultFetchPolicy():r,a=o.initialFetchPolicy;Object.assign(o,{initialFetchPolicy:void 0===a?i:a,fetchPolicy:"standby"})}else o.fetchPolicy||(o.fetchPolicy=(null===(t=this.observable)||void 0===t?void 0:t.options.initialFetchPolicy)||this.getDefaultFetchPolicy());return o},e.prototype.getDefaultFetchPolicy=function(){var e,t;return(null===(e=this.queryHookOptions.defaultOptions)||void 0===e?void 0:e.fetchPolicy)||(null===(t=this.client.defaultOptions.watchQuery)||void 0===t?void 0:t.fetchPolicy)||"cache-first"},e.prototype.onCompleted=function(e){},e.prototype.onError=function(e){},e.prototype.useObservableQuery=function(){var e=this.observable=this.renderPromises&&this.renderPromises.getSSRObservable(this.watchQueryOptions)||this.observable||this.client.watchQuery(or(this.queryHookOptions.defaultOptions,this.watchQueryOptions));this.obsQueryFields=(0,b.useMemo)(function(){return{refetch:e.refetch.bind(e),reobserve:e.reobserve.bind(e),fetchMore:e.fetchMore.bind(e),updateQuery:e.updateQuery.bind(e),startPolling:e.startPolling.bind(e),stopPolling:e.stopPolling.bind(e),subscribeToMore:e.subscribeToMore.bind(e)}},[e]);var t=!(!1===this.queryHookOptions.ssr||this.queryHookOptions.skip);return this.renderPromises&&t&&(this.renderPromises.registerSSRObservable(e),e.getCurrentResult().loading&&this.renderPromises.addObservableQueryPromise(e)),e},e.prototype.setResult=function(e){var t=this.result;t&&t.data&&(this.previousData=t.data),this.result=e,this.forceUpdate(),this.handleErrorOrCompleted(e)},e.prototype.handleErrorOrCompleted=function(e){!e.loading&&(e.error?this.onError(e.error):e.data&&this.onCompleted(e.data))},e.prototype.getCurrentResult=function(){return this.result||this.handleErrorOrCompleted(this.result=this.observable.getCurrentResult()),this.result},e.prototype.toQueryResult=function(e){var t=this.toQueryResultCache.get(e);if(t)return t;var n=e.data,o=(e.partial,ez(e,["data","partial"]));return this.toQueryResultCache.set(e,t=eT(eT(eT({data:n},o),this.obsQueryFields),{client:this.client,observable:this.observable,variables:this.observable.variables,called:!0,previousData:this.previousData})),!t.error&&n9(e.errors)&&(t.error=new iO({graphQLErrors:e.errors})),t},e.prototype.unsafeHandlePartialRefetch=function(e){e.partial&&this.queryHookOptions.partialRefetch&&!e.loading&&(!e.data||0===Object.keys(e.data).length)&&"cache-only"!==this.observable.options.fetchPolicy&&(Object.assign(e,{loading:!0,networkStatus:m.refetch}),this.observable.refetch())},e}(),i7=o$({uri:"https://graphql.contentful.com/content/v1/spaces/kp51zybwznx4/environments/master",credentials:"same-origin",headers:{Authorization:"Bearer dpk-7L7rGYzkKk-jZwtIDnyhui6DgLq6VTapJNI7W44"},fetch:y.fetch});function ae(e,t,n){let{currentLocale:o,isPreview:r,isSSR:i}=t;return i6(e,{...n,context:{clientName:"global",...n?.context},variables:{locale:o,preview:r,...n?.variables},ssr:i??n?.ssr})}ew(ae,"useGlobalComponentsContentfulQuery");var at=(0,b.createContext)({isPreview:!1,isSSR:!1,currentLocale:"en-US"}),an=ew(({component:e})=>(console.error(`${e} not implemented! This must be specified via DesignSystemContext.`),null),"NotImplemented"),ao=(0,b.createContext)({buttonComponent:()=>(0,C.tZ)(an,{component:"buttonComponent"}),localeDropdownComponent:()=>(0,C.tZ)(an,{component:"localeDropdownComponent"}),modalComponent:()=>(0,C.tZ)(an,{component:"modalComponent"}),sectionComponent:()=>(0,C.tZ)(an,{component:"sectionComponent"}),toggleComponent:()=>(0,C.tZ)(an,{component:"toggleComponent"})}),ar={kind:"Document",definitions:[{kind:"OperationDefinition",operation:"query",name:{kind:"Name",value:"CookieModalQuery"},variableDefinitions:[{kind:"VariableDefinition",variable:{kind:"Variable",name:{kind:"Name",value:"preview"}},type:{kind:"NonNullType",type:{kind:"NamedType",name:{kind:"Name",value:"Boolean"}}}},{kind:"VariableDefinition",variable:{kind:"Variable",name:{kind:"Name",value:"locale"}},type:{kind:"NonNullType",type:{kind:"NamedType",name:{kind:"Name",value:"String"}}}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"cookieModalCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"preview"},value:{kind:"Variable",name:{kind:"Name",value:"preview"}}},{kind:"Argument",name:{kind:"Name",value:"locale"},value:{kind:"Variable",name:{kind:"Name",value:"locale"}}},{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"1"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"CookieModalAll"}}]}}]}}]}},{kind:"FragmentDefinition",name:{kind:"Name",value:"CookieModalAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"CookieModal"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"landingScreenTitle"}},{kind:"Field",name:{kind:"Name",value:"content"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"json"}}]}},{kind:"Field",name:{kind:"Name",value:"acceptAllButtonText"}},{kind:"Field",name:{kind:"Name",value:"essentialOnlyButtonText"}},{kind:"Field",name:{kind:"Name",value:"settingsButtonText"}},{kind:"Field",name:{kind:"Name",value:"settingsScreenTitle"}},{kind:"Field",name:{kind:"Name",value:"cookieCategoriesCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"10"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"CookieCategoryAll"}}]}}]}},{kind:"Field",name:{kind:"Name",value:"acceptSelectedButtonText"}},{kind:"Field",name:{kind:"Name",value:"saveChangesText"}},{kind:"Field",name:{kind:"Name",value:"changesSavedText"}},{kind:"Field",name:{kind:"Name",value:"toggleEnabledText"}},{kind:"Field",name:{kind:"Name",value:"toggleDisabledText"}}]}},{kind:"FragmentDefinition",name:{kind:"Name",value:"CookieCategoryAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"CookieCategory"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"description"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"json"}}]}},{kind:"Field",name:{kind:"Name",value:"categoryCookieName"}},{kind:"Field",name:{kind:"Name",value:"displayMode"}},{kind:"Field",name:{kind:"Name",value:"isEssential"}},{kind:"Field",name:{kind:"Name",value:"enableToggle"}},{kind:"Field",name:{kind:"Name",value:"cookiesCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"200"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"TrackingCookieAll"}}]}}]}}]}},{kind:"FragmentDefinition",name:{kind:"Name",value:"TrackingCookieAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"TrackingCookie"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"name"}},{kind:"Field",name:{kind:"Name",value:"provider"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"json"}}]}},{kind:"Field",name:{kind:"Name",value:"domains"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"json"}}]}},{kind:"Field",name:{kind:"Name",value:"purpose"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"json"}}]}},{kind:"Field",name:{kind:"Name",value:"expiration"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"json"}}]}},{kind:"Field",name:{kind:"Name",value:"pattern"}}]}}]},ai=new Set(["localhost","sc-corp.net","appspot.com"]),aa=class{constructor(e,t){ex(this,"client"),ex(this,"initializeClient",ew((e,t)=>{let n=new v.Graphene,o=this.getGrapheneFlavor(t),r=this.getGrapheneFriendlyString(t),i=(0,F.getWebConfig)({partitionName:e,flavor:o,variant:r});return n.initialize({networkHandler:new _.BrowserNetworkHandler(i),debugMode:!1,logTimeInterval:5e3}),n},"initializeClient")),ex(this,"getGrapheneFlavor",ew(e=>{let t=eE(e);return ai.has(t)?"development":"production"},"getGrapheneFlavor")),ex(this,"getGrapheneFriendlyString",ew(e=>e?e.replace(/[\s.]+/g,"_"):"Unknown","getGrapheneFriendlyString")),ex(this,"logMetric",ew((e,t)=>{this.client.increment({metricsName:e,dimensions:t})},"logMetric")),ex(this,"logError",ew((e,t,n)=>{let o={description:this.getGrapheneFriendlyString(t),error:this.getGrapheneFriendlyString(n.name)};this.client.increment({metricsName:`error_${e}`,dimensions:o})},"logError")),this.client=this.initializeClient(e,t)}};ew(aa,"BrowserGrapheneClient");var as=ew(e=>N.get(e),"getCookie"),al=ew((e,t,n="")=>{N.set(e,t,{expires:365,secure:!0,domain:n})},"setCookie"),ac=ew(e=>e||eE((document?.location?.hostname??"").toLowerCase()),"getCookieDomain"),au=ew((e,t)=>{for(let n in t)al(n,t[n].toString(),e)},"setOptInCookies"),ad=ew(e=>{let t={};for(let n of e){let{categoryCookieName:e}=n,o=as(e);o&&(t[e]="true"===o)}return t},"getAllCookies"),ah=ew(e=>{let t={};for(let n of e){let{categoryCookieName:e}=n;t[e]=!0}return t},"acceptAllCookies"),ap=ew(e=>{let t={};for(let n of e){let{categoryCookieName:e,isEssential:o}=n;t[e]=o}return t},"essentialOnlyCookies"),af=ew((e,t)=>({...ap(e),...t}),"acceptCustomCookies"),am=ew(e=>{let t=new Map;for(let n of e){let e=new Set,o=n.categoryCookieName;n.cookiesCollection.items.forEach(({name:t,pattern:n})=>{let o=n?new RegExp(n):void 0;e.add({name:t,regex:o})}),t.set(o,e)}return t},"createCookieMapping"),ag=ew((e,t,n)=>{if("sc-cookies-accepted"===e)return;let o=n.get(e);if(o)for(let e of o)if(e.regex){let n=Object.keys(t).find(t=>e.regex.test(t));if(!n)continue;N.remove(n)}else e.name in t&&N.remove(e.name)},"removeRelevantCookies"),av=ew((e,t)=>{let n=N.get();for(let o in e)e[o]||ag(o,n,t)},"removeCookiesForNonacceptedCategories"),ab=ew(e=>{let t=N.get(),n=new Set,o=[];for(let[t,r]of e)for(let{name:e,regex:t}of r)t?o.push(t):n.add(e);let r=[];for(let e in t)!n.has(e)&&(o.some(t=>t.test(e))||r.push(e));return r},"auditForUncategorizedCookies"),ay=ew(async()=>{let e=await fetch("https://www.snapchat.com/cookies/api/is_cookie_popup_eligible");if(200!==e.status){let{status:t,statusText:n}=e;throw Error(`${t} : ${n}`)}return(await e.json()).popupEnabled},"fetchShouldDisplayModal"),ak=ew(async()=>{let e=await fetch("https://www.snapchat.com/cookies/api/user_location");if(200!==e.status){let{status:t,statusText:n}=e;throw Error(`${t} : ${n}`)}let t=await e.json(),n=t.country,o=t.region;return`${n}-${o}`},"fetchUserLocation"),aw=ew(e=>{let{buttonComponent:t}=(0,b.useContext)(ao);return t(e)},"DesignAgnosticButton"),ax=ew(e=>{let{localeDropdownComponent:t}=(0,b.useContext)(ao);return t(e)},"DesignAgnosticLocaleDropdown"),aS=ew(e=>{let{modalComponent:t}=(0,b.useContext)(ao);return t(e)},"DesignAgnosticModal"),aC=ew(e=>{let{sectionComponent:t}=(0,b.useContext)(ao);return t(e)},"DesignAgnosticSection"),aF=ew(e=>{let{toggleComponent:t}=(0,b.useContext)(ao);return t(e)},"DesignAgnosticToggle"),a_=class extends b.Component{constructor(e){super(e);let{partition:t,hostname:n}=e,o=t&&new aa(t,n);this.state={hasError:!1,grapheneClient:o}}componentDidCatch(e){this.setState({hasError:!0}),this.state.grapheneClient?.logError(this.props.component,"unhandled",e),this.props.onError&&this.props.onError(e)}render(){let{children:e,renderInstead:t}=this.props,{hasError:n}=this.state;return n?t??null:e}};function aN(e,t,n){let o=e.displayName??"UnknownComponent";return ew(r=>{let i=(document?.location?.hostname??"unknown").toLowerCase(),{onError:a}=r;return(0,C.tZ)(a_,{hostname:i,partition:t,component:o,onError:a,renderInstead:n,children:(0,C.tZ)(e,{...r})})},"ComponentWithBrowserErrorBoundary")}ew(a_,"BrowserErrorBoundary"),ew(aN,"withErrorBoundary");var a$=ew(({document:e,className:t})=>{let{currentLocale:n}=(0,b.useContext)(at),o=ew(e=>{let t=new URL(e);return t.searchParams.has("lang")||t.searchParams.set("lang",n),t.href},"renderUri");return(0,C.tZ)("div",{"data-testid":"mwp-cookie-rich-text",className:(0,E.cx)("contentful-rich-text",t),dangerouslySetInnerHTML:{__html:(0,$.S)(e,{renderNode:{hyperlink:e=>{let t=e.content.map(e=>{if(e&&"text"===e.nodeType)return e.value},[]).join(""),n=o(e.data.uri);return`<a href="${n}" target="_blank" rel="noopener">${t}</a>`}}})}})},"CookieRichText"),aE=ew(({children:e})=>(0,C.tZ)("div",{className:"modal-footer",children:e}),"CookieModalFooter"),aO=ew(({className:e,width:t,height:n,backgroundType:o="White"})=>(0,C.tZ)("svg",{version:"1.1",xmlns:"http://www.w3.org/2000/svg",x:"0px",y:"0px",viewBox:"0 0 500 500",className:e,width:t,height:n,children:(0,C.tZ)("g",{id:"Layer_1",children:(0,C.BX)("g",{children:[(0,C.tZ)("g",{children:(0,C.tZ)("path",{style:{fill:"Black"!==o?"#FFFFFF":void 0},d:"M484.6,369.3c-2-6.8-11.9-11.6-11.9-11.6l0,0c-0.9-0.5-1.7-0.9-2.4-1.3c-16.4-7.9-30.8-17.4-43.1-28.2\n                c-9.8-8.7-18.2-18.2-25-28.4c-8.2-12.4-12.1-22.8-13.8-28.4c-0.9-3.7-0.8-5.1,0-7c0.6-1.6,2.5-3.1,3.4-3.8\n                c5.5-3.9,14.4-9.7,19.9-13.2c4.7-3.1,8.8-5.7,11.2-7.4c7.7-5.4,12.9-10.8,16-16.7c4-7.6,4.5-16,1.4-24.3\n                c-4.2-11.1-14.6-17.8-27.8-17.8c-2.9,0-6,0.3-9,1c-7.6,1.6-14.8,4.3-20.8,6.7c-0.4,0.2-0.9-0.2-0.9-0.6\n                c0.6-14.9,1.3-34.9-0.3-53.9c-1.5-17.2-5-31.7-10.8-44.3c-5.8-12.7-13.4-22.1-19.3-28.9c-5.7-6.5-15.5-16-30.5-24.5\n                c-21-12-45-18.1-71.1-18.1c-26.1,0-50,6.1-71.1,18.1c-15.8,9-25.9,19.2-30.5,24.5c-5.9,6.8-13.5,16.2-19.3,28.9\n                c-5.8,12.6-9.3,27.1-10.8,44.3c-1.6,19.1-1,37.5-0.3,53.9c0,0.5-0.5,0.8-0.9,0.6c-6-2.3-13.2-5-20.7-6.7c-3-0.7-6-1-9-1\n                c-13.2,0-23.6,6.7-27.8,17.8c-3.1,8.3-2.7,16.7,1.4,24.3c3.1,5.9,8.4,11.4,16,16.7c2.4,1.7,6.4,4.3,11.2,7.4\n                c5.3,3.5,14,9.1,19.5,13c0.7,0.5,3,2.3,3.8,4.1c0.8,2,0.9,3.4-0.1,7.3c-1.7,5.7-5.6,15.9-13.7,28.1c-6.8,10.2-15.2,19.7-25,28.4\n                C60.5,339,46,348.5,29.7,356.5c-0.8,0.4-1.7,0.8-2.7,1.4l0,0c0,0-9.8,5-11.6,11.4c-2.7,9.5,4.5,18.4,11.9,23.2\n                c12.1,7.8,26.9,12,35.4,14.3c2.4,0.6,4.5,1.2,6.5,1.8c1.2,0.4,4.3,1.6,5.6,3.3c1.7,2.1,1.9,4.8,2.5,7.8v0c0.9,5,3,11.2,9.2,15.5\n                c6.8,4.7,15.5,5,26.4,5.5c11.5,0.4,25.8,1,42.1,6.4c7.6,2.5,14.4,6.7,22.4,11.6c16.6,10.2,37.2,22.9,72.5,22.9\n                c35.3,0,56.1-12.8,72.8-23c7.9-4.8,14.7-9,22.1-11.5c16.3-5.4,30.6-5.9,42.1-6.4c11-0.4,19.6-0.7,26.4-5.5\n                c6.6-4.6,8.6-11.4,9.4-16.6c0.5-2.5,0.8-4.8,2.3-6.7c1.2-1.6,4.1-2.7,5.4-3.2c2-0.6,4.2-1.2,6.7-1.9c8.6-2.3,19.3-5,32.3-12.4\n                C485.2,385.6,486.3,374.7,484.6,369.3z"})}),(0,C.tZ)("path",{style:{fill:"Black"===o?"#FFFFFF":void 0},d:"M498.2,364c-3.5-9.5-10.1-14.5-17.6-18.7c-1.4-0.8-2.7-1.5-3.8-2c-2.2-1.2-4.5-2.3-6.8-3.5\n              c-23.5-12.5-41.8-28.2-54.6-46.8c-4.3-6.3-7.3-12-9.4-16.6c-1.1-3.1-1-4.9-0.3-6.5c0.6-1.2,2.2-2.5,3-3.1c4-2.7,8.2-5.4,11-7.2\n              c5-3.3,9-5.8,11.6-7.6c9.6-6.7,16.4-13.9,20.6-21.9c5.9-11.3,6.7-24.2,2.1-36.3c-6.4-16.8-22.3-27.3-41.5-27.3\n              c-4,0-8,0.4-12.1,1.3c-1.1,0.2-2.1,0.5-3.1,0.7c0.2-11.4-0.1-23.6-1.1-35.6c-3.6-42-18.3-64-33.7-81.6c-6.4-7.3-17.5-18-34.2-27.6\n              C305.1,10.6,278.8,3.9,250,3.9c-28.7,0-55,6.7-78.3,20c-16.8,9.6-27.9,20.3-34.3,27.6c-15.3,17.5-30,39.6-33.7,81.6\n              c-1,11.9-1.3,24.1-1.1,35.6c-1-0.3-2.1-0.5-3.1-0.7c-4-0.9-8.1-1.3-12.1-1.3c-19.3,0-35.2,10.4-41.5,27.3\n              c-4.6,12.1-3.8,25,2.1,36.3c4.2,8,11,15.2,20.6,21.9c2.6,1.8,6.6,4.4,11.6,7.6c2.7,1.8,6.7,4.3,10.6,6.9c0.6,0.4,2.7,1.9,3.4,3.4\n              c0.8,1.7,0.8,3.5-0.4,6.8c-2.1,4.6-5,10.1-9.2,16.3c-12.4,18.2-30.3,33.6-53,46c-12,6.4-24.6,10.6-29.9,25\n              c-4,10.9-1.4,23.2,8.8,33.6l0,0c3.3,3.6,7.5,6.8,12.8,9.7c12.4,6.9,23,10.2,31.3,12.5c1.5,0.4,4.8,1.5,6.3,2.8\n              c3.7,3.2,3.2,8.1,8.1,15.2c3,4.4,6.4,7.4,9.2,9.4c10.3,7.1,21.9,7.6,34.2,8c11.1,0.4,23.7,0.9,38.1,5.7c6,2,12.1,5.8,19.3,10.2\n              c17.2,10.6,40.8,25,80.2,25c39.4,0,63.1-14.5,80.5-25.2c7.1-4.4,13.3-8.1,19-10c14.4-4.7,27-5.2,38.1-5.7\n              c12.3-0.5,23.9-0.9,34.2-8c3.2-2.2,7.3-5.9,10.5-11.5c3.5-6,3.4-10.2,6.8-13.2c1.4-1.2,4.3-2.2,5.9-2.7\n              c8.4-2.3,19.1-5.7,31.7-12.6c5.6-3.1,10-6.5,13.4-10.3c0,0,0.1-0.1,0.1-0.1C499.7,386.6,502.1,374.6,498.2,364z M463.2,382.8\n              c-21.4,11.8-35.6,10.5-46.6,17.7c-9.4,6-3.8,19.1-10.7,23.8c-8.4,5.8-33.2-0.4-65.1,10.2c-26.4,8.7-43.2,33.8-90.7,33.8\n              c-47.6,0-64-25-90.7-33.8c-32-10.6-56.8-4.4-65.1-10.2c-6.8-4.7-1.3-17.7-10.7-23.8c-11-7.1-25.3-5.9-46.6-17.7\n              c-13.6-7.5-5.9-12.2-1.4-14.4c77.4-37.5,89.8-95.4,90.3-99.7c0.7-5.2,1.4-9.3-4.3-14.6c-5.5-5.1-30.1-20.3-36.9-25\n              c-11.3-7.9-16.2-15.7-12.6-25.4c2.5-6.7,8.8-9.2,15.4-9.2c2,0,4.1,0.2,6.1,0.7c12.4,2.7,24.4,8.9,31.3,10.6c1,0.2,1.8,0.3,2.6,0.3\n              c3.7,0,5-1.9,4.8-6.1c-0.8-13.5-2.7-39.9-0.6-64.5c2.9-33.9,13.9-50.7,26.9-65.6c6.2-7.1,35.5-38.1,91.6-38.1\n              c56.2,0,85.3,30.9,91.6,38.1c13,14.9,23.9,31.7,26.9,65.6c2.1,24.6,0.3,51-0.6,64.5c-0.3,4.5,1.1,6.1,4.8,6.1\n              c0.8,0,1.6-0.1,2.6-0.3c6.9-1.7,19-7.9,31.3-10.6c2-0.4,4.1-0.7,6.1-0.7c6.6,0,12.8,2.5,15.4,9.2c3.7,9.7-1.3,17.5-12.6,25.4\n              c-6.8,4.7-31.3,19.9-36.9,25c-5.7,5.3-5,9.4-4.3,14.6c0.6,4.4,12.9,62.2,90.3,99.7C469.1,370.6,476.8,375.3,463.2,382.8z"})]})})}),"GhostLogo"),aD=ew(({backgroundType:e,supportedLocales:t,onLocaleChange:n})=>{let{currentLocale:o}=(0,b.useContext)(at),r=(0,b.useRef)(null),i=ew(()=>r.current,"containerProvider");return(0,C.BX)("div",{"data-testid":"mwp-cookie-modal-header",className:"modal-header",ref:r,children:[(0,C.tZ)("div",{className:"logo-container",children:(0,C.tZ)(aO,{width:24,height:24,backgroundType:e})}),(0,C.tZ)("div",{className:"locale-container",children:(0,C.tZ)(ax,{supportedLocales:t,currentLocale:t[o],onLocaleChange:n,containerProvider:i})})]})},"CookieModalHeader"),aT=ew(({backgroundType:e,supportedLocales:t,title:n,content:o,settingsButtonText:r,acceptAllButtonText:i,essentialOnlyButtonText:a,onSettingsbuttonClick:s,onAcceptAllButtonClick:l,onAcceptEssentialButtonClick:c,onLocaleChange:u})=>(0,C.BX)("div",{"data-testid":"mwp-cookie-landing-screen",className:"cookie-landing-screen",children:[(0,C.tZ)(aD,{backgroundType:e,supportedLocales:t,onLocaleChange:u}),(0,C.BX)("div",{"data-testid":"mwp-cookie-modal-body",className:"modal-body",children:[(0,C.tZ)("h2",{className:"cookie-title",children:n}),(0,C.tZ)(a$,{document:o,className:"landing-content"})]}),(0,C.BX)(aE,{children:[(0,C.tZ)(aw,{isPrimary:!0,text:a,onClick:c}),(0,C.tZ)(aw,{isPrimary:!0,text:i,onClick:l}),(0,C.tZ)(aw,{isPrimary:!1,text:r,onClick:s})]})]}),"CookieLandingScreen"),az=ew(({title:e,cookieCategories:t,categoriesState:n,updateCategoriesState:o,activeToggleLabel:r,inactiveToggleLabel:i,isModal:a})=>(0,C.BX)(C.HY,{children:[(0,C.tZ)("h2",{className:"cookie-title",children:e}),t.map(e=>{if(a&&"Settings Only"===e.displayMode)return null;let t=i;return e.isEssential&&!e.enableToggle&&(t=r),n[e.categoryCookieName]&&(t=r),(0,C.tZ)("div",{"data-testid":"mwp-cookie-category-description",children:(0,C.BX)("div",{className:"category-border",children:[(0,C.BX)("div",{className:"category-title-container",children:[(0,C.tZ)("h5",{className:"category-title",children:e.title}),(0,C.tZ)("p",{className:"category-status",children:t}),!0===e.enableToggle&&(0,C.tZ)("div",{className:"category-toggle",children:(0,C.tZ)(aF,{id:`${e.sys.id}-slider`,isChecked:!!n[e.categoryCookieName],onToggle:()=>o(e.categoryCookieName,!n[e.categoryCookieName])})})]}),(0,C.tZ)(a$,{className:"category-description",document:e.description.json})]})},e.sys.id)})]}),"CookieCategories"),aM=ew(({backgroundType:e,supportedLocales:t,onLocaleChange:n,title:o,cookieCategories:r,activeToggleLabel:i,inactiveToggleLabel:a,categoriesState:s,updateCategoriesState:l,acceptAllButtonText:c,essentialOnlyButtonText:u,acceptSelectedButtonText:d,onAcceptAllButtonClick:h,onAcceptEssentialButtonClick:p,onAcceptSelectedButtonClick:f})=>(0,C.BX)("div",{"data-testid":"mwp-cookie-modal-settings-screen",className:"cookie-settings-screen",children:[(0,C.tZ)(aD,{backgroundType:e,supportedLocales:t,onLocaleChange:n}),(0,C.tZ)("div",{className:"modal-body",children:(0,C.tZ)(az,{title:o,cookieCategories:r,categoriesState:s,updateCategoriesState:l,activeToggleLabel:i,inactiveToggleLabel:a,isModal:!0})}),(0,C.BX)(aE,{children:[(0,C.tZ)(aw,{isPrimary:!0,text:u,onClick:p}),(0,C.tZ)(aw,{isPrimary:!0,text:c,onClick:h}),(0,C.tZ)(aw,{isPrimary:!0,text:d,onClick:f})]})]}),"CookieSettingsScreen"),aI="CookieModal",aP=ew(({supportedLocales:e,cookieDomain:t,portalRoot:n,backgroundType:o="White",onLocaleChange:r,onComplete:i,onEvent:a,forceVisible:s})=>{let l=(0,b.useContext)(at),{data:c}=ae(ar,l,{client:l.client}),u=w(c?.cookieModalCollection.items),[d,h]=(0,b.useMemo)(()=>u?.cookieCategoriesCollection?.items?.length?[u.cookieCategoriesCollection.items.map(e=>{let{categoryCookieName:t,displayMode:n,enableToggle:o,isEssential:r}=e;return{categoryCookieName:t,displayMode:n,enableToggle:o,isEssential:r}}),am(u.cookieCategoriesCollection.items)]:[],[u]),[p]=(0,b.useState)(ac(t)),[f,m]=(0,b.useState)(),[g,y]=(0,b.useState)("Unknown"),[C,F]=(0,b.useState)(!1),[_,N]=(0,b.useState)({}),$=ew((e,t)=>{let n=k(_);n[e]=t,N(n)},"updateCategoryState"),[E,D]=(0,b.useState)(!1),[T,z]=(0,b.useState)("landing"),[M,I]=(0,b.useState)(),P=(0,b.useCallback)(e=>{if(!ai.has(p))for(let t of ab(e)){let e={cookieName:t,userLocation:g};M?.logMetric("modal_uncategorized_cookie",e)}},[M,p,g]),j=(0,b.useCallback)((e,t)=>{h&&d&&(au(p,ew((e,t={})=>S(e,(e,n)=>x(t[n])||t[n]!==e),"filterToChanges")(e,t)),av(e,h),P(h),i?.({didUserInteract:C,userLocation:g,cookieAcceptance:e}),F(!1))},[p,h,d,C,i,g,F,P]);(0,b.useEffect)(()=>{let e=window.location.hostname;I(new aa(v.PARTITION.COOKIE_MODAL_COMPONENTS,e))},[I]),(0,b.useEffect)(()=>{M&&ew(async()=>{try{let e=await ay();m(e)}catch(t){let e=eC(t);M.logError(aI,"shouldDisplayModal",e),m(!0)}},"getIsInModalRequiredRegion")()},[M,m]),(0,b.useEffect)(()=>{M&&ew(async()=>{try{let e=await ak();y(e)}catch(t){let e=eC(t);M.logError(aI,"userLocation",e)}},"fetchUserRegion")()},[M,y]),(0,b.useEffect)(()=>{d&&N(ap(d))},[d,N]),(0,b.useEffect)(()=>{if(x(f)||!h||!d||E)return;D(!0);let e=ad(d);if(e["sc-cookies-accepted"]){j({...f?ap(d):ah(d),...e},e);return}f?F(!0):j(ah(d))},[E,f,h,d,D,F,j]);let R=ew(()=>{if(!d)return;let e=ah(d);j(e),L("accept_all",e)},"acceptAll"),L=ew((e,t)=>{a?.({component:aI,action:"Click",label:e});let n={preferences:`${t.Preferences??!1}`,performance:`${t.Performance??!1}`,marketing:`${t.Marketing??!1}`,userLocation:g};M?.logMetric(`${e}_clicks_modal`,n)},"logUserAction"),A=ew(()=>{if(!d)return;let e=ap(d);j(e),L("accept_essential",e)},"acceptEssential"),V=ew(()=>{if(!d)return;let e=af(d,_);j(e),L("accept_selected",e)},"acceptSelected");return u?(0,O.jsxs)(aS,{isDisplayed:s??C,backgroundType:o,portalRoot:n,children:["landing"===T&&(0,O.jsx)(aT,{backgroundType:o,supportedLocales:e,title:u.landingScreenTitle,content:u.content.json,settingsButtonText:u.settingsButtonText,onSettingsbuttonClick:()=>z("settings"),acceptAllButtonText:u.acceptAllButtonText,onAcceptAllButtonClick:R,essentialOnlyButtonText:u.essentialOnlyButtonText,onAcceptEssentialButtonClick:A,onLocaleChange:r}),"settings"===T&&(0,O.jsx)(aM,{backgroundType:o,supportedLocales:e,title:u.landingScreenTitle,cookieCategories:u.cookieCategoriesCollection.items,activeToggleLabel:u.toggleEnabledText,inactiveToggleLabel:u.toggleDisabledText,categoriesState:_,updateCategoriesState:$,acceptAllButtonText:u.acceptAllButtonText,essentialOnlyButtonText:u.essentialOnlyButtonText,acceptSelectedButtonText:u.acceptSelectedButtonText,onAcceptAllButtonClick:R,onAcceptEssentialButtonClick:A,onAcceptSelectedButtonClick:V,onLocaleChange:r})]}):null},"CookieModal");aP.displayName=aI;var aj=aN(aP,v.PARTITION.COOKIE_MODAL_COMPONENTS),aR=ew(({children:e,client:t,...n})=>{let{isPreview:o=!1,isSSR:r=!1,currentLocale:i="en-US"}=n,a=(0,O.jsx)(at.Provider,{value:{isPreview:o,isSSR:r,currentLocale:i,client:t},children:(0,O.jsx)(ao.Provider,{value:n,children:e})});if(t)return a;let s=new iG({link:i7,cache:new iN});return(0,O.jsx)(iK,{client:s,children:a})},"CookieProvider"),aL="en-US",aA={[aL]:{code:aL,name:"English (United States)"}},aV={currentLocale:aL,supportedLocales:aA,onError:console.error,isPreview:!1,isSSR:!1,isUrlCurrent:ew(e=>{let t=new URL(e,window.location.href);return window.location.hostname===t.hostname&&window.location.pathname!==t.pathname},"defaultIsUrlCurrent")},aq=(0,b.createContext)({onError:e=>console.error(e),currentLocale:aL,hostname:"unknown",supportedLocales:aA}),aZ=ew(({children:e,value:t})=>(0,C.tZ)(aq.Provider,{value:D(aV,t),children:e}),"GlobalComponentsContextProvider"),aB=ew((e,t,n)=>{aG({"gtm.start":new Date().getTime(),event:"gtm.js"});let o=e.createElement("script");o.async=!0,o.src=`https://www.googletagmanager.com/gtm.js?id=${t}`,n&&(o.nonce=n),e.head.appendChild(o)},"googleTagDataLayerSet"),aQ=ew((e,t)=>{if(""===e)return;let n=aH(e);window[`ga-disable-${n}`]=!t},"toggleGoogleTracking"),aH=ew(e=>e.includes("UA-")||e.includes("G-")?e:`UA-${e}`,"getGaidWithPrefix"),aG=ew(e=>{window.dataLayer=window.dataLayer??[],window.dataLayer.push(e)},"configureDataLayer"),aU=ew((e,t,n)=>{window&&(aQ(t,!0),aB(document,e,n))},"enableGoogleTagManager"),aW=ew(e=>{(window.ga.q=window.ga.q??[]).push(...e)},"windowGaArgumentSetter"),aK=ew(e=>{window.GoogleAnalyticsObject="ga",window.ga=window.ga??aW,window.ga.l=new Date().getTime;let t=document.createElement("script");t.async=!0,t.src="https://www.google-analytics.com/analytics.js",e&&(t.nonce=e),document.head.appendChild(t)},"googleAnalyticsArgumentsSet"),aY=ew((e,t=[],n)=>{if(window){if(aQ(e,!0),e.includes("UA-"))aK(n),ga("create",e,"auto"),t.forEach(e=>{ga("require",e.name,e.options)}),ga("set","anonymizeIp",!0),ga("send","pageview");else{window.dataLayer=window.dataLayer??[];let t=document.createElement("script");t.async=!0,t.src=`https://www.googletagmanager.com/gtag/js?id=${e}`,t.onload=()=>{aG({js:new Date,config:e})},n&&(t.nonce=n),document.head.appendChild(t)}}},"enableGoogleAnalytics"),aX=ew((e,t,n,o)=>{let r=aH(t);e&&t?aU(e,r,o):t&&aY(r,n,o)},"enablePerformanceAnalytics"),aJ=((l=aJ||{}).DarkMode="Black",l.LightMode="White",l),a0=class{constructor(e){ex(this,"lowEntropyHints"),ex(this,"highEntropyHints"),ex(this,"getLowEntropyHints",ew(()=>(this.lowEntropyHints||(this.lowEntropyHints=Object.freeze(this.computeLowEntropyHints())),this.lowEntropyHints),"getLowEntropyHints")),ex(this,"getHighEntropyHintsAsync",ew(async e=>{this.highEntropyHints||(this.highEntropyHints={});let t=e.hints.filter(e=>!this.highEntropyHints[e]);if(0===t.length)return this.highEntropyHints;let n=await this.computeHighEntropyHintsAsync(...t),o=k(this.highEntropyHints);return T(o,n),this.highEntropyHints=o,Object.freeze(this.highEntropyHints)},"getHighEntropyHintsAsync")),ex(this,"getCachedHighEntropyHints",ew(()=>this.highEntropyHints,"getCachedHighEntropyHints")),this.lowEntropyHints=e?.lowEntropyHints,this.highEntropyHints=e?.highEntropyHints}};function a1(e){if(e.browsers.find(({brand:e})=>"Edge"===e))return!1;if(e.browsers.find(({brand:e,majorVersion:t})=>("Chrome"===e||"Chromium"===e)&&t>=85)||e.browsers.find(({brand:e,majorVersion:t})=>"Firefox"===e&&t>=113))return!0;let t=e.browsers.find(({brand:e})=>"Opera Mini"===e);return!!(e.browsers.find(({brand:e,majorVersion:t})=>"Opera"===e&&t>=71)&&!t||e.browsers.find(({brand:e,majorVersion:t})=>"Safari"===e&&t>=17))}function a2(e){return!!(e.browsers.find(({brand:e,majorVersion:t})=>("Chromium"===e||"Chrome"===e)&&t>=32)||e.browsers.find(({brand:e})=>"Opera Mini"===e)||e.browsers.find(({brand:e,majorVersion:t})=>"Firefox"===e&&t>=65)||e.browsers.find(({brand:e,majorVersion:t})=>"Safari"===e&&t>=16))}ew(a0,"AbstractBrowserFeature"),ew(a1,"supportsAvif"),ew(a2,"supportsWebP");var a3={isMobile:!1,platform:"Unknown",browsers:[],saveData:!1},a5=class extends a0{computeLowEntropyHints(){return this.lowEntropyHints??a3}computeHighEntropyHintsAsync(){return Promise.resolve(this.highEntropyHints??{})}};ew(a5,"StaticBrowserFeature");var a4=(0,b.createContext)(new a5({lowEntropyHints:{isMobile:!0,saveData:!1,browsers:[],platform:"Unknown"}}));a4.Provider;var a6=ew(e=>e.getLowEntropyHints().isMobile,"isMobileOs"),a8={Top:"Top",Middle:"Middle",Bottom:"Bottom"},a9={Black:"Black",White:"White",Yellow:"Yellow",Gray:"Gray",TransparentLight:"TransparentLight",TransparentDark:"TransparentDark"};a9.Black,a9.White,a9.Yellow,a9.Gray,a9.TransparentLight,a9.TransparentDark;var a7="#FFF",se=ew(({min:e,max:t})=>{if(!z(e)&&!z(t)){if(t<e)throw Error(`max=${t} is less than min=${e}`);return`@media screen and (min-width: ${e}px) and (max-width: ${t}px)`}return!z(e)&&z(t)?`@media screen and (min-width: ${e}px)`:z(e)&&!z(t)?`@media screen and (max-width: ${t}px)`:se({min:0})},"mediaQueryForRange"),st=se({max:768}),sn=se({min:769});se({min:1241}),se({min:1921});var so="@media screen and (max-width: 480px)",sr="@media screen and (max-width: 1024px)",si="@media screen and (min-width: 1025px)",sa={None:"None",Laptop:"Laptop",Phone:"Phone",Shadow:"Shadow"},ss={Desktop:"Desktop",Mobile:"Mobile"},sl={Compact:"Compact",Regular:"Regular",Large:"Large",Flat:"Flat"};function sc(e){return`var(${e})`}ew(sc,"m"),sc("not-a-variable");var su={"--accordion-header-padding":sc("--spacing-m"),"--accordion-header-desktop-font-size":sc("--h6-desktop-font-size"),"--accordion-header-mobile-font-size":sc("--h6-mobile-font-size"),"--accordion-header-desktop-font-line-height":sc("--h6-desktop-font-line-height"),"--accordion-header-mobile-font-line-height":sc("--h6-mobile-font-line-height"),"--accordion-header-desktop-font-weight":sc("--h6-desktop-font-weight"),"--accordion-header-mobile-font-weight":sc("--h6-mobile-font-weight")},sd={"--accordion-bg-color":"#000","--accordion-divider-border-color":"#C7C7CC","--accordion-header-color":"#FFF",...su},sh={"--accordion-bg-color":"#FFFC00","--accordion-divider-border-color":"#53575B","--accordion-header-color":"#000",...su},sp={"--accordion-bg-color":"#FFF","--accordion-divider-border-color":"#D4D5D6","--accordion-header-color":"#000",...su},sf={"--accordion-bg-color":"#F0F1F2","--accordion-divider-border-color":"#C7C7CC","--accordion-header-color":"#000",...su},sm={"--banner-fg-color":"#000000","--banner-font-size":"14px","--banner-font-line-height":"18px"},sg={"--banner-bg-color":"#FFFC00",...sm},sv={...sm,"--banner-bg-color":"#000000","--banner-fg-color":a7},sb={"--banner-bg-color":a7,...sm},sy={"--banner-bg-color":"#F0F1F2",...sm},sk={"--block-title-color":sc("--foreground-color"),"--block-subtitle-color":sc("--foreground-color"),"--block-eyebrow-color":sc("--foreground-color"),"--block-title-desktop-font-size":sc("--h2-desktop-font-size"),"--block-title-desktop-font-stretch":sc("--h2-desktop-font-stretch"),"--block-title-desktop-font-line-height":sc("--h2-desktop-font-line-height"),"--block-title-mobile-font-size":sc("--h2-mobile-font-size"),"--block-header-d-padding":`${sc("--spacing-m")} ${sc("--spacing-xl")}`,"--block-header-m-padding":sc("--spacing-m"),"--block-boundary-d-padding":sc("--spacing-xl"),"--block-boundary-m-padding":sc("--spacing-m")},sw={"--break-total-desktop-height":"96px","--break-half-desktop-height":"48px","--break-total-mobile-height":"32px","--break-half-mobile-height":"16px"},sx={"--button-desktop-font-size":sc("--action-desktop-font-size"),"--button-desktop-font-line-height":sc("--action-desktop-font-line-height"),"--button-desktop-font-weight":sc("--action-desktop-font-weight"),"--button-mobile-font-size":sc("--action-mobile-font-size"),"--button-mobile-font-line-height":sc("--action-mobile-font-line-height"),"--button-mobile-font-weight":sc("--action-mobile-font-weight")},sS={"--button-regular-padding":"11px 31px","--button-compact-padding":"7px 15px","--button-border-width":"1px","--button-border-radius":"64px","--button-hover-shadow":sc("--box-shadow-xl"),"--button-active-shadow":sc("--box-shadow-m")},sC={"--button-primary-bg-color":"#FFFC00","--button-primary-hover-bg-color":"#FFFC00","--button-primary-border-color":"#FFFC00","--button-primary-hover-border-color":"#FFFC00","--button-primary-fg-color":"#000","--button-secondary-bg-color":"#FFF","--button-secondary-hover-bg-color":"#FFF","--button-secondary-border-color":"#FFF","--button-secondary-hover-border-color":"#FFF","--button-secondary-fg-color":"#000","--button-flat-fg-color":"#FFF",...sx,...sS},sF={"--button-primary-bg-color":"#000","--button-primary-hover-bg-color":"#000","--button-primary-border-color":"#000","--button-primary-hover-border-color":"#000","--button-primary-fg-color":"#FFF","--button-secondary-bg-color":"#FFF","--button-secondary-hover-bg-color":"#FFF","--button-secondary-border-color":"#FFF","--button-secondary-hover-border-color":"#FFF","--button-secondary-fg-color":"#000","--button-flat-fg-color":"#000",...sx,...sS},s_={"--button-primary-bg-color":"#FFFC00","--button-primary-hover-bg-color":"#FFFC00","--button-primary-border-color":"#FFFC00","--button-primary-hover-border-color":"#FFFC00","--button-primary-fg-color":"#000","--button-secondary-bg-color":"#000","--button-secondary-hover-bg-color":"#000","--button-secondary-border-color":"#000","--button-secondary-hover-border-color":"#000","--button-secondary-fg-color":"#FFF","--button-flat-fg-color":"#000",...sx,...sS},sN={"--button-primary-bg-color":"#FFFC00","--button-primary-hover-bg-color":"#FFFC00","--button-primary-border-color":"#FFFC00","--button-primary-hover-border-color":"#FFFC00","--button-primary-fg-color":"#000","--button-secondary-bg-color":"#FFF","--button-secondary-hover-bg-color":"#FFF","--button-secondary-border-color":"#FFF","--button-secondary-hover-border-color":"#FFF","--button-secondary-fg-color":"#000","--button-flat-fg-color":"#000",...sx,...sS},s$={"--content-desktop-grid-gap":sc("--spacing-l"),"--content-mobile-grid-gap":sc("--spacing-m")},sE={"--dropdown-menu-padding":sc("--spacing-m"),"--dropdown-item-fg-color":sc("--action-default-color"),"--dropdown-item-fg-hover-color":sc("--action-hover-color"),"--dropdown-item-fg-active-color":sc("--action-active-color")},sO={"--dropdown-menu-bg-color":a7,"--dropdown-item-bg-hover-color":"#F0F1F2","--dropdown-item-bg-active-color":"#F7F8F9","--dropdown-menu-border-color":"#E9EAEB",...sE},sD={"--dropdown-menu-bg-color":"#121314","--dropdown-item-bg-hover-color":"#53575B","--dropdown-item-bg-active-color":"#3A3E41","--dropdown-menu-border-color":"#000",...sE},sT={"--form-grid-gap":sc("--spacing-xs")},sz={"--form-input-placeholder-color":"#858D94","--form-input-fg-color":"#000","--form-input-error-color":"#C50A33","--form-input-bg-color":"#FFF","--form-input-border-color":"#E9EAEB","--form-input-active-border-color":"#E9EAEB","--form-input-border-width":"1px","--form-input-border-radius":sc("--spacing-s"),"--form-input-desktop-font-size":sc("--h6-desktop-font-size"),"--form-input-desktop-font-line-height":sc("--h6-desktop-font-line-height"),"--form-input-desktop-font-weight":sc("--h6-desktop-font-weight"),"--form-input-mobile-font-size":sc("--h6-mobile-font-size"),"--form-input-mobile-font-line-height":sc("--h6-mobile-font-line-height"),"--form-input-mobile-font-weight":sc("--h6-mobile-font-weight"),"--form-input-box-shadow":sc("--box-shadow-m"),"--form-input-padding":`0 ${sc("--spacing-l")}`,"--form-input-mobile-font-stretch":sc("--h6-mobile-font-stretch"),"--form-input-desktop-font-stretch":sc("--h6-desktop-font-stretch")},sM={...sT,...sz},sI={"--global-header-bg-color":"#000","--global-header-nav-screen-bg-color":"#000","--global-header-fg-color":"#C7C7CC","--global-header-item-color":sc("--action-default-color"),"--global-header-item-hover-color":sc("--action-hover-color"),"--global-header-item-active-color":sc("--action-active-color"),"--global-header-navigator-item-color":"#D4D5D6","--global-header-navigator-item-hover-color":"#E9EAEB","--global-header-navigator-item-active-color":a7};({...sI});var sP={"--global-header-bg-color":"#FFF","--global-header-nav-screen-bg-color":"#FFF","--global-header-fg-color":"#3A3E41","--global-header-item-color":sc("--action-default-color"),"--global-header-item-hover-color":sc("--action-hover-color"),"--global-header-item-active-color":sc("--action-active-color"),"--global-header-navigator-item-color":"#3A3E41","--global-header-navigator-item-hover-color":"#000000","--global-header-navigator-item-active-color":"#000000"};({...sP});var sj={"--hero-title-color":sc("--foreground-color"),"--hero-subtitle-color":sc("--foreground-color"),"--hero-text-desktop-padding":sc("--spacing-xxl"),"--hero-text-mobile-padding":"0","--hero-boundary-desktop-padding":`${sc("--spacing-xl")} ${sc("--spacing-xxxl")}`,"--hero-boundary-mobile-padding":`${sc("--spacing-xxl")} ${sc("--spacing-xl")}`,"--hero-desktop-grid-gap":"0","--hero-mobile-grid-gap":sc("--spacing-xl")},sR={"--hyperlink-color":sc("--action-default-color"),"--hyperlink-hover-color":sc("--action-hover-color"),"--hyperlink-desktop-font-size":sc("--action-desktop-font-size"),"--hyperlink-desktop-font-line-height":sc("--action-desktop-font-line-height"),"--hyperlink-desktop-font-weight":sc("--action-desktop-font-weight"),"--hyperlink-desktop-font-text-decoration":sc("--action-desktop-font-text-decoration"),"--hyperlink-mobile-font-size":sc("--action-mobile-font-size"),"--hyperlink-mobile-font-line-height":sc("--action-mobile-font-line-height"),"--hyperlink-mobile-font-weight":sc("--action-mobile-font-weight"),"--hyperlink-mobile-font-text-decoration":sc("--action-mobile-font-text-decoration")},sL={"--icon-color":sc("--foreground-color")},sA={"--icon-button-border-width":"2px"},sV={...sA,"--icon-button-fg-color":"#3A3E41","--icon-button-bg-color":"#FFF","--icon-button-border-color":"#3A3E41","--icon-button-hover-bg-color":"#FFF","--icon-button-hover-border-color":"#FCF000","--icon-button-disabled-bg-color":"#3A3E41","--icon-button-disabled-fg-color":"#000000","--icon-button-disabled-border-color":"#121314"},sq={...sA,"--icon-button-fg-color":"#FFF","--icon-button-bg-color":"#121314","--icon-button-border-color":"#FFF","--icon-button-hover-bg-color":"#121314","--icon-button-hover-border-color":"#FCF000","--icon-button-disabled-bg-color":"#F0F1F2","--icon-button-disabled-fg-color":"#858D94","--icon-button-disabled-border-color":"#C7C7CC"},sZ={"--mosaic-border-radius":sc("--border-radius-m"),"--mosaic-grid-gap":sc("--spacing-l"),"--mosaic-title-color":"#FFF","--mosaic-highlight-color":"#FFFC00","--mosaic-duration-color":"#FFF"},sB={"--mosaic-border-radius":sc("--border-radius-m"),"--mosaic-grid-gap":sc("--spacing-l"),"--mosaic-title-color":"#FFF","--mosaic-highlight-color":"#FFFC00","--mosaic-duration-color":"#FFF"},sQ={"--pagination-text-active-color":"#000","--pagination-text-color":"#3A3E41","--pagination-text-hover-color":"#C7C7CC"},sH={"--quote-bg-color":"#F0F1F2","--quote-fg-color":"#000","--quote-author-desktop-font-size":"16px","--quote-author-desktop-font-weight":"500","--quote-author-mobile-font-size":"16px","--quote-author-mobile-font-weight":"500"},sG={"--quote-bg-color":"#FFF","--quote-fg-color":"#000","--quote-author-desktop-font-size":"16px","--quote-author-desktop-font-weight":"500","--quote-author-mobile-font-size":"16px","--quote-author-mobile-font-weight":"500"},sU={"--side-navigation-active-link-color":"#FFFC00"},sW={"--sub-navigation-item-color":sc("--action-default-color"),"--sub-navigation-item-hover-color":sc("--action-hover-color"),"--sub-navigation-item-active-color":sc("--action-active-color"),"--sub-navigation-item-hover-decoration-color":"#C7C7CC"},sK={...sW,"--sub-navigation-item-active-decoration-color":"#FFFC00"},sY={...sW,"--sub-navigation-item-active-decoration-color":"#000000"},sX={...sW,"--sub-navigation-item-active-decoration-color":"#000000"},sJ={...sW,"--sub-navigation-item-active-decoration-color":"#000000"},s0={"--tabs-item-color":sc("--action-default-color"),"--tabs-item-hover-color":sc("--action-hover-color"),"--tabs-item-active-color":sc("--action-active-color")},s1={"--tabs-underline-color":"#D4D5D6",...s0},s2={"--tabs-underline-color":"#3A3E41",...s0},s3=["sdsm-default","sdsm-secondary","sdsm-tertiary","sdsm-quaternary","sdsm-quinary"],s5={},s4={name:"DefaultMotif",fontFamily:"Graphik","sdsm-default":{name:"Yellow background",legacyName:a9.Yellow,sdsm:{"--box-shadow-xs":"0px 2px 4px 0 rgba(0, 0, 0, 0.12)","--box-shadow-s":"0px 4px 8px 0 rgba(0, 0, 0, 0.12)","--box-shadow-m":"0px 8px 16px 0 rgba(0, 0, 0, 0.12)","--box-shadow-l":"0px 12px 20px 0 rgba(0, 0, 0, 0.12)","--box-shadow-xl":"0px 16px 32px 0 rgba(0, 0, 0, 0.12)","--border-radius-xs":"1px","--border-radius-s":"4px","--border-radius-m":"8px","--border-radius-l":"16px","--border-radius-xl":"24px","--spacing-xxxs":"2px","--spacing-xxs":"4px","--spacing-xs":"8px","--spacing-s":"12px","--spacing-m":"16px","--spacing-l":"24px","--spacing-xl":"32px","--spacing-xxl":"40px","--spacing-xxxl":"48px","--spacing-xxxxl":"64px","--font-family":"Graphik","--text-desktop-font-size":"18px","--text-desktop-font-line-height":"24px","--text-desktop-font-weight":"400","--text-mobile-font-size":"16px","--text-mobile-font-line-height":"120%","--text-mobile-font-weight":"400","--h1-desktop-font-size":"64px","--h1-desktop-font-line-height":"120%","--h1-desktop-font-weight":"600","--h1-desktop-font-stretch":"normal","--h1-mobile-font-size":"46px","--h1-mobile-font-line-height":"120%","--h1-mobile-font-weight":"600","--h1-mobile-font-stretch":"normal","--h2-desktop-font-size":"48px","--h2-desktop-font-line-height":"120%","--h2-desktop-font-weight":"600","--h2-desktop-font-stretch":"normal","--h2-mobile-font-size":"32px","--h2-mobile-font-line-height":"120%","--h2-mobile-font-weight":"600","--h2-mobile-font-stretch":"normal","--h3-desktop-font-size":"40px","--h3-desktop-font-line-height":"120%","--h3-desktop-font-weight":"600","--h3-desktop-font-stretch":"normal","--h3-mobile-font-size":"28px","--h3-mobile-font-line-height":"120%","--h3-mobile-font-weight":"600","--h3-mobile-font-stretch":"normal","--h4-desktop-font-size":"28px","--h4-desktop-font-line-height":"120%","--h4-desktop-font-weight":"500","--h4-desktop-font-stretch":"normal","--h4-mobile-font-size":"24px","--h4-mobile-font-line-height":"120%","--h4-mobile-font-weight":"500","--h4-mobile-font-stretch":"normal","--h5-desktop-font-size":"24px","--h5-desktop-font-line-height":"120%","--h5-desktop-font-weight":"500","--h5-desktop-font-stretch":"normal","--h5-mobile-font-size":"20px","--h5-mobile-font-line-height":"120%","--h5-mobile-font-weight":"500","--h5-mobile-font-stretch":"normal","--h6-desktop-font-size":"18px","--h6-desktop-font-line-height":"120%","--h6-desktop-font-weight":"500","--h6-desktop-font-stretch":"normal","--h6-mobile-font-size":"18px","--h6-mobile-font-line-height":"120%","--h6-mobile-font-weight":"500","--h6-mobile-font-stretch":"normal","--p1-desktop-font-size":"20px","--p1-desktop-font-line-height":"160%","--p1-desktop-font-weight":"400","--p1-desktop-font-stretch":"normal","--p1-mobile-font-size":"18px","--p1-mobile-font-line-height":"160%","--p1-mobile-font-weight":"400","--p1-mobile-font-stretch":"normal","--p2-desktop-font-size":"16px","--p2-desktop-font-line-height":"160%","--p2-desktop-font-weight":"400","--p2-desktop-font-stretch":"normal","--p2-mobile-font-size":"16px","--p2-mobile-font-line-height":"160%","--p2-mobile-font-weight":"400","--p2-mobile-font-stretch":"normal","--p3-desktop-font-size":"14px","--p3-desktop-font-line-height":"160%","--p3-desktop-font-weight":"400","--p3-desktop-font-stretch":"normal","--p3-mobile-font-size":"14px","--p3-mobile-font-line-height":"160%","--p3-mobile-font-weight":"400","--p3-mobile-font-stretch":"normal","--p4-desktop-font-size":"12px","--p4-desktop-font-line-height":"160%","--p4-desktop-font-weight":"400","--p4-desktop-font-stretch":"normal","--p4-mobile-font-size":"12x","--p4-mobile-font-line-height":"160%","--p4-mobile-font-weight":"400","--p4-mobile-font-stretch":"normal","--action-desktop-font-size":"16px","--action-desktop-font-line-height":"24px","--action-desktop-font-weight":"500","--action-mobile-font-size":"16px","--action-mobile-font-line-height":"120%","--action-mobile-font-weight":"500","--annotation-desktop-font-size":"14px","--annotation-desktop-font-weight":"500","--annotation-desktop-font-line-height":"20px","--annotation-desktop-font-letter-spacing":"-.14px","--annotation-mobile-font-size":"14px","--annotation-mobile-font-weight":"500","--annotation-mobile-font-line-height":"20px","--annotation-mobile-font-letter-spacing":"-.14px","--background-color":"#FFFC00","--foreground-color":"#000","--action-default-color":"#3A3E41","--action-hover-color":"#121314","--action-active-color":"#000","--action-desktop-font-text-decoration":"none","--action-mobile-font-text-decoration":"none"},"sdsm-button":sF,"sdsm-header":sP,"sdsm-block":sk,"sdsm-block-boundary":s5,"sdsm-block-split-panel":s5,"sdsm-detail-summary":s5,"sdsm-tab":s2,"sdsm-hero":sj,"sdsm-page":s5,"sdsm-content":s$,"sdsm-toggle-slider":{"--toggle-slider-background-color":"#D4D5D6","--toggle-slider-active-color":"#000000","--toggle-slider-switch-color":a7},"sdsm-break":sw,"sdsm-quote":sG,"sdsm-code":s5,"sdsm-filter-dropdown-menu":s5,"sdsm-footer":{"--footer-bg-color":"#FFFC00","--footer-border-color":"#53575B","--footer-divider-border-color":"#53575B","--footer-bar-bg-color":"#FFFC00","--footer-bar-divider-border-color":"#53575B"},"sdsm-dropdown-menu":sO,"sdsm-image-button":s5,"sdsm-table":s5,"sdsm-accordion":sh,"sdsm-footnote":{"--footnote-fg-color":"#858D94","--footnote-bg-color":"#FFF"},"sdsm-tile":{"--tile-background-color":"#FFF","--tile-foreground-color":"#000"},"sdsm-banner":sg,"sdsm-hyperlink":sR,"sdsm-search":{"--search-no-results-bg-color":"#F0F1F2","--search-no-results-fg-color":"#121314"},"sdsm-pagination":sQ,"sdsm-snapchat-embed":s5,"sdsm-side-navigation":{"--side-navigation-active-link-color":"#000"},"sdsm-modal":{"--modal-background-color":"rgba(255, 255, 255, .7)","--modal-close-background-color":"#000","--modal-close-foreground-color":"#FFF"},"sdsm-mosaic":sZ,"sdsm-spinner":{"--spinner-fg-color":"#000"},"sdsm-icon-button":sq,"sdsm-tooltip":s5,"sdsm-icon":sL,"sdsm-form":sM,"sdsm-sub-navigation":sY,"sdsm-progress-bar":{"--progress-bar-background-color":"#FFF","--progress-bar-progress-color":"#C7C7CC"},"sdsm-chart-toggle":{"--chart-toggle-buttons-bg-color":a7,"--chart-toggle-buttons-color":"#53575B","--chart-toggle-buttons-border-color":"#53575B","--chart-toggle-buttons-active-bg-color":"#000000","--chart-toggle-buttons-active-color":a7,"--chart-toggle-buttons-active-border-color":"#000000"},"sdsm-block-navigation":{"--block-navigation-buttons-bg-color":a7,"--block-navigation-buttons-color":"#53575B","--block-navigation-buttons-border-color":"#53575B","--block-navigation-buttons-active-color":a7,"--block-navigation-buttons-active-border-color":"#000000","--block-navigation-buttons-active-bg-color":"#000000"},"sdsm-bar-chart":s5,"sdsm-line-chat":s5,"sdsm-geo-map":s5},"sdsm-secondary":{name:"Black background",legacyName:a9.Black,sdsm:{"--background-color":"#000","--foreground-color":"#FFF","--action-default-color":"#C7C7CC","--action-hover-color":"#D4D5D6","--action-active-color":"#FFF","--box-shadow-xs":"0px 2px 4px 0 rgba(255, 255, 255, 0.12)","--box-shadow-s":"0px 4px 8px 0 rgba(255, 255, 255, 0.12)","--box-shadow-m":"0px 8px 16px 0 rgba(255, 255, 255, 0.12)","--box-shadow-l":"0px 12px 20px 0 rgba(255, 255, 255, 0.12)","--box-shadow-xl":"0px 16px 32px 0 rgba(255, 255, 255, 0.12)"},"sdsm-button":sC,"sdsm-header":sI,"sdsm-block":sk,"sdsm-tab":s1,"sdsm-toggle-slider":{"--toggle-slider-background-color":"#F0F1F2","--toggle-slider-active-color":"#FFFC00","--toggle-slider-switch-color":a7},"sdsm-quote":sH,"sdsm-footer":{"--footer-bg-color":"#000000","--footer-border-color":"#53575B","--footer-divider-border-color":"#53575B","--footer-bar-bg-color":"#000","--footer-bar-divider-border-color":"#53575B"},"sdsm-accordion":sd,"sdsm-dropdown-menu":sD,"sdsm-banner":sv,"sdsm-hyperlink":sR,"sdsm-pagination":{"--pagination-text-active-color":"#FFF","--pagination-text-color":"#FFF","--pagination-text-hover-color":"#E9EAEB"},"sdsm-side-navigation":sU,"sdsm-hero":sj,"sdsm-modal":{"--modal-background-color":"rgba(0, 0, 0, .7)","--modal-close-background-color":"#FFF","--modal-close-foreground-color":"#000"},"sdsm-spinner":{"--spinner-fg-color":"#FFF"},"sdsm-sub-navigation":sK,"sdsm-progress-bar":{"--progress-bar-background-color":"#D4D5D6","--progress-bar-progress-color":"#FFFC00"},"sdsm-chart-toggle":{"--chart-toggle-buttons-bg-color":a7,"--chart-toggle-buttons-color":"#53575B","--chart-toggle-buttons-border-color":"#53575B","--chart-toggle-buttons-active-bg-color":"#FFFC00","--chart-toggle-buttons-active-color":"#000000","--chart-toggle-buttons-active-border-color":"#FFFC00"},"sdsm-block-navigation":{"--block-navigation-buttons-color":"#53575B","--block-navigation-buttons-active-color":"#000000","--block-navigation-buttons-bg-color":a7,"--block-navigation-buttons-border-color":"#53575B","--block-navigation-buttons-active-border-color":"#FFFC00","--block-navigation-buttons-active-bg-color":"#FFFC00"},"sdsm-break":sw,"sdsm-content":s$,"sdsm-mosaic":sB,"sdsm-icon":sL,"sdsm-icon-button":sV,"sdsm-form":sM},"sdsm-tertiary":{name:"White background",legacyName:a9.White,sdsm:{"--background-color":"#FFF","--foreground-color":"#000","--action-default-color":"#3A3E41","--action-hover-color":"#121314","--action-active-color":"#000"},"sdsm-button":s_,"sdsm-header":sP,"sdsm-block":sk,"sdsm-tab":s2,"sdsm-toggle-slider":{"--toggle-slider-background-color":"#53575B","--toggle-slider-active-color":"#FFFC00","--toggle-slider-switch-color":a7},"sdsm-quote":sH,"sdsm-footer":{"--footer-bg-color":"#FFF","--footer-border-color":"#C7C7CC","--footer-divider-border-color":"#C7C7CC","--footer-bar-bg-color":"#F0F1F2","--footer-bar-divider-border-color":"#F0F1F2"},"sdsm-accordion":sp,"sdsm-dropdown-menu":sO,"sdsm-banner":sb,"sdsm-hyperlink":sR,"sdsm-pagination":sQ,"sdsm-side-navigation":sU,"sdsm-sub-navigation":sJ,"sdsm-progress-bar":{"--progress-bar-background-color":"#D4D5D6","--progress-bar-progress-color":"#FFFC00"},"sdsm-chart-toggle":{"--chart-toggle-buttons-bg-color":a7,"--chart-toggle-buttons-color":"#53575B","--chart-toggle-buttons-border-color":"#53575B","--chart-toggle-buttons-active-bg-color":"#FFFC00","--chart-toggle-buttons-active-color":"#000000","--chart-toggle-buttons-active-border-color":"#FFFC00"},"sdsm-block-navigation":{"--block-navigation-buttons-bg-color":a7,"--block-navigation-buttons-color":"#53575B","--block-navigation-buttons-border-color":"#53575B","--block-navigation-buttons-active-color":"#000000","--block-navigation-buttons-active-border-color":"#FFFC00","--block-navigation-buttons-active-bg-color":"#FFFC00"},"sdsm-break":sw,"sdsm-hero":sj,"sdsm-content":s$,"sdsm-mosaic":sZ,"sdsm-icon":sL,"sdsm-icon-button":sq,"sdsm-form":sM},"sdsm-quaternary":{name:"Gray background",legacyName:a9.Gray,sdsm:{"--background-color":"#F0F1F2","--foreground-color":"#000","--action-default-color":"#3A3E41","--action-hover-color":"#121314","--action-active-color":"#000"},"sdsm-button":sN,"sdsm-header":sP,"sdsm-block":sk,"sdsm-tab":s2,"sdsm-toggle-slider":{"--toggle-slider-background-color":"#000000","--toggle-slider-active-color":"#FFFC00","--toggle-slider-switch-color":a7},"sdsm-quote":sG,"sdsm-footer":{"--footer-bg-color":"#F0F1F2","--footer-border-color":"#C7C7CC","--footer-divider-border-color":"#C7C7CC","--footer-bar-bg-color":"#F0F1F2","--footer-bar-divider-border-color":"#C7C7CC"},"sdsm-accordion":sf,"sdsm-dropdown-menu":sO,"sdsm-banner":sy,"sdsm-hyperlink":sR,"sdsm-pagination":sQ,"sdsm-side-navigation":sU,"sdsm-sub-navigation":sX,"sdsm-progress-bar":{"--progress-bar-background-color":"#FFF","--progress-bar-progress-color":"#FFFC00"},"sdsm-chart-toggle":{"--chart-toggle-buttons-bg-color":a7,"--chart-toggle-buttons-color":"#53575B","--chart-toggle-buttons-border-color":"#53575B","--chart-toggle-buttons-active-bg-color":"#000000","--chart-toggle-buttons-active-color":a7,"--chart-toggle-buttons-active-border-color":"#000000"},"sdsm-block-navigation":{"--block-navigation-buttons-bg-color":a7,"--block-navigation-buttons-color":"#53575B","--block-navigation-buttons-border-color":"#53575B","--block-navigation-buttons-active-color":a7,"--block-navigation-buttons-active-border-color":"#000000","--block-navigation-buttons-active-bg-color":"#000000"},"sdsm-break":sw,"sdsm-content":s$,"sdsm-hero":sj,"sdsm-mosaic":sZ,"sdsm-icon":sL,"sdsm-icon-button":sq,"sdsm-form":sM}},s6=ew(e=>{if(e)return Object.fromEntries(Object.entries(e).map(([e,t])=>[`data-${M(e)}`,t]))},"dataSetToAttributes"),s8=ew(()=>!!("undefined"!=typeof window&&window.document&&window.document.createElement),"isBrowser");function s9(e,...t){return String.raw({raw:e},...t)}ew(s9,"globalCss");var s7=ew(e=>e?`background--${M(e)}`:void 0,"getBackgroundClassName"),le=ew(e=>`var(${e})`,"cssVar");function lt(e=1e3){let t=s8(),{getCachedHighEntropyHints:n}=(0,b.useContext)(a4),o=(0,b.useCallback)(()=>({width:t?window.innerWidth:n()?.viewportWidth,height:t?window.innerHeight:n()?.viewportHeight}),[n,t]),[r,i]=(0,b.useState)(o),a=(0,b.useCallback)(()=>{i(o())},[o,i]);return(0,b.useEffect)(()=>{if(!t)return;a();let n=I(a,e,{leading:!1,trailing:!0});return window.addEventListener("resize",n),()=>window.removeEventListener("resize",n)},[a,t,e]),r}function ln(){return s4.name}function lo(e){let t=new Map;for(let n of s3){let o=s4[n]?.[e];if(!o)continue;t.has(o)||t.set(o,new Set);let r=t.get(o),i=s7(s4[n].legacyName);"sdsm-default"===n&&r.add(".sdsm"),r.add(`.${n}`),r.add(`.${i}`)}let n={};for(let[e,o]of t.entries())n[[...o].join(",\n")]=e;(0,E.injectGlobal)(n)}function lr(e,t){let n=window.document.createElement("style");t&&Object.entries(t).forEach(([e,t])=>{n.dataset[e]=t}),n.innerHTML=e,window.document.head.appendChild(n)}ew(lt,"useWindowSize"),"undefined"!=typeof window&&window.document,ew(ln,"getMotifName"),ew(lo,"injectStyles"),ew(function(e,t){let n="";for(let[e,o]of Object.entries(t)){for(let[t,r]of(n+=`${e} {
`,Object.entries(o)))n+=`  ${t}: ${r};
`;n+="}\n"}lr(n,{source:"motif",motif:s4.name,component:e})},"customInjectGlobal"),ew(lr,"injestGlobalStylesheet");var li={injectedStyles:new Set},la=(0,b.createContext)(li);function ls(e){let t=ln(),{injectedStyles:n}=(0,b.useContext)(la),o=`${t}-${e}`;n.has(o)||(n.add(o),lo(e))}ew(ls,"useMotifStyles");var ll=s9`
  /* Required to be listed. I.e. body.sdsm { font-family: ... } */
  font-family: ${sc("--font-family")}, Helvetica, Arial, sans-serif;

  ${sn} {
    font-size: ${sc("--text-desktop-font-size")};
    line-height: ${sc("--text-desktop-font-line-height")};
    font-weight: ${sc("--text-desktop-font-weight")};
  }

  ${st} {
    font-size: ${sc("--text-mobile-font-size")};
    line-height: ${sc("--text-mobile-font-line-height")};
    font-weight: ${sc("--text-mobile-font-weight")};
  }
`,lc=s9`
  ${sn} {
    font-size: ${sc("--h1-desktop-font-size")};
    line-height: ${sc("--h1-desktop-font-line-height")};
    font-weight: ${sc("--h1-desktop-font-weight")};
    font-stretch: ${sc("--h1-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--h1-mobile-font-size")};
    line-height: ${sc("--h1-mobile-font-line-height")};
    font-weight: ${sc("--h1-mobile-font-weight")};
    font-stretch: ${sc("--h1-mobile-font-stretch")};
  }
`,lu=s9`
  ${sn} {
    font-size: ${sc("--h2-desktop-font-size")};
    line-height: ${sc("--h2-desktop-font-line-height")};
    font-weight: ${sc("--h2-desktop-font-weight")};
    font-stretch: ${sc("--h2-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--h2-mobile-font-size")};
    line-height: ${sc("--h2-mobile-font-line-height")};
    font-weight: ${sc("--h2-mobile-font-weight")};
    font-stretch: ${sc("--h2-mobile-font-stretch")};
  }
`,ld=s9`
  ${sn} {
    font-size: ${sc("--h3-desktop-font-size")};
    line-height: ${sc("--h3-desktop-font-line-height")};
    font-weight: ${sc("--h3-desktop-font-weight")};
    font-stretch: ${sc("--h3-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--h3-mobile-font-size")};
    line-height: ${sc("--h3-mobile-font-line-height")};
    font-weight: ${sc("--h3-mobile-font-weight")};
    font-stretch: ${sc("--h3-mobile-font-stretch")};
  }
`,lh=s9`
  ${sn} {
    font-size: ${sc("--h4-desktop-font-size")};
    line-height: ${sc("--h4-desktop-font-line-height")};
    font-weight: ${sc("--h4-desktop-font-weight")};
    font-stretch: ${sc("--h4-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--h4-mobile-font-size")};
    line-height: ${sc("--h4-mobile-font-line-height")};
    font-weight: ${sc("--h4-mobile-font-weight")};
    font-stretch: ${sc("--h4-mobile-font-stretch")};
  }
`,lp=s9`
  ${sn} {
    font-size: ${sc("--h5-desktop-font-size")};
    line-height: ${sc("--h5-desktop-font-line-height")};
    font-weight: ${sc("--h5-desktop-font-weight")};
    font-stretch: ${sc("--h5-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--h5-mobile-font-size")};
    line-height: ${sc("--h5-mobile-font-line-height")};
    font-weight: ${sc("--h5-mobile-font-weight")};
    font-stretch: ${sc("--h5-mobile-font-stretch")};
  }
`,lf=s9`
  ${sn} {
    font-size: ${sc("--h6-desktop-font-size")};
    line-height: ${sc("--h6-desktop-font-line-height")};
    font-weight: ${sc("--h6-desktop-font-weight")};
    font-stretch: ${sc("--h6-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--h6-mobile-font-size")};
    line-height: ${sc("--h6-mobile-font-line-height")};
    font-weight: ${sc("--h6-mobile-font-weight")};
    font-stretch: ${sc("--h6-mobile-font-stretch")};
  }
`;s9`
  .${"sdsm"},
  .${"sdsm"} * {
      box-sizing: border-box;
  }

  .${"sdsm"} { ${ll} }
  .${"sdsm"} h1, h1.${"sdsm"} { ${lc} }
  .${"sdsm"} h2, h2.${"sdsm"} { ${lu} }
  .${"sdsm"} h3, h3.${"sdsm"} { ${ld} }
  .${"sdsm"} h4, h4.${"sdsm"} { ${lh} }
  .${"sdsm"} h5, h5.${"sdsm"} { ${lp} }
  .${"sdsm"} h6, h6.${"sdsm"} { ${lf} }
`,(0,E.css)(ll),(0,E.css)(lc),(0,E.css)(lu),(0,E.css)(lh),(0,E.css)(lh);var lm=(0,E.css)(lp);(0,E.css)(lf);var lg=(0,E.css)`
  ${sn} {
    font-size: ${sc("--p1-desktop-font-size")};
    line-height: ${sc("--p1-desktop-font-line-height")};
    font-weight: ${sc("--p1-desktop-font-weight")};
    font-stretch: ${sc("--p1-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--p1-mobile-font-size")};
    line-height: ${sc("--p1-mobile-font-line-height")};
    font-weight: ${sc("--p1-mobile-font-weight")};
    font-stretch: ${sc("--p1-mobile-font-stretch")};
  }
`;(0,E.css)`
  ${sn} {
    font-size: ${sc("--p2-desktop-font-size")};
    line-height: ${sc("--p2-desktop-font-line-height")};
    font-weight: ${sc("--p2-desktop-font-weight")};
    font-stretch: ${sc("--p2-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--p2-mobile-font-size")};
    line-height: ${sc("--p2-mobile-font-line-height")};
    font-weight: ${sc("--p2-mobile-font-weight")};
    font-stretch: ${sc("--p2-mobile-font-stretch")};
  }
`,(0,E.css)`
  ${sn} {
    font-size: ${sc("--p3-desktop-font-size")};
    line-height: ${sc("--p3-desktop-font-line-height")};
    font-weight: ${sc("--p3-desktop-font-weight")};
    font-stretch: ${sc("--p3-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--p3-mobile-font-size")};
    line-height: ${sc("--p3-mobile-font-line-height")};
    font-weight: ${sc("--p3-mobile-font-weight")};
    font-stretch: ${sc("--p3-mobile-font-stretch")};
  }
`,(0,E.css)`
  ${sn} {
    font-size: ${sc("--p4-desktop-font-size")};
    line-height: ${sc("--p4-desktop-font-line-height")};
    font-weight: ${sc("--p4-desktop-font-weight")};
    font-stretch: ${sc("--p4-desktop-font-stretch")};
  }

  ${st} {
    font-size: ${sc("--p4-mobile-font-size")};
    line-height: ${sc("--p4-mobile-font-line-height")};
    font-weight: ${sc("--p4-mobile-font-weight")};
    font-stretch: ${sc("--p4-mobile-font-stretch")};
  }
`;var lv=(0,E.css)`
  color: ${sc("--action-default-color")};
  :hover {
    color: ${sc("--action-hover-color")};
  }
  :active {
    color: ${sc("--action-active-color")};
  }

  ${st} {
    font-size: ${sc("--action-mobile-font-size")};
    line-height: ${sc("--action-mobile-font-line-height")};
    font-weight: ${sc("--action-mobile-font-weight")};
  }

  ${sn} {
    font-size: ${sc("--action-desktop-font-size")};
    line-height: ${sc("--action-desktop-font-line-height")};
    font-weight: ${sc("--action-desktop-font-weight")};
  }
`,lb={initialValue:"off",onToggle:P,transitionDurationMs:250},ly=ew(e=>{let{initialValue:t,onToggle:n,transitionDurationMs:o}={...lb,...e},[r,i]=(0,b.useState)(t),a=(0,b.useCallback)((e,t)=>{e!==t&&(i(e),n(e,t))},[n,i]);(0,b.useEffect)(()=>{let e;return"turning-off"===r&&(e=setTimeout(a.bind(void 0,"off","turning-off"),o)),"turning-on"===r&&(e=setTimeout(a.bind(void 0,"on","turning-on"),o)),()=>{e&&clearTimeout(e)}},[r,o,a]);let s=ew(()=>{"off"===r&&a("turning-on",r)},"turnOn"),l=ew(()=>{"on"===r&&a("turning-off",r)},"turnOff"),c=ew((e="toggle")=>{"on"===e?s():"off"===e?l():"turning-off"===r||"off"===r?s():l()},"toggle");return{state:r,toggle:c}},"useToggleState"),lk={chart:ew(e=>(0,C.tZ)("svg",{width:"16",height:"16",viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e,children:(0,C.tZ)("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M9.57768 6.37882L12.6856 2.89522L11.9394 2.22949L9.35981 5.12089L6.25449 3.42708L3.11646 7.17908L3.88353 7.82063L6.4955 4.69763L9.57768 6.37882ZM11.4375 5.15611H13.1875V13.9999H11.4375V5.15611ZM5.6875 7.01548H7.4375V13.9999H5.6875V7.01548ZM4.5625 9.31236H2.8125V13.9999H4.5625V9.31236ZM8.5625 8.09361H10.3125V13.9999H8.5625V8.09361Z"})}),"ChartIcon"),flashlight:ew(e=>(0,C.tZ)("svg",{width:"16",height:"16",viewBox:"0 0 16 16",...e,xmlns:"http://www.w3.org/2000/svg",children:(0,C.tZ)("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M13.15 0.20C13.70 0.43 14.27 0.88 14.77 1.47L14.47 1.77L14.77 1.47C15.28 2.07 15.63 2.73 15.79 3.34C15.94 3.92 15.93 4.59 15.49 5.01L15.22 4.68L15.49 5.01L8.62 11.58C8.69 11.84 8.74 12.09 8.75 12.33C8.77 12.66 8.74 12.99 8.57 13.28L8.55 13.31L8.52 13.35L6.53 15.59L6.52 15.61L6.50 15.62C6.24 15.87 5.91 15.96 5.58 15.95C5.24 15.93 4.89 15.83 4.53 15.66C3.81 15.32 3.03 14.68 2.31 13.84C1.62 13.01 0.94 12.04 0.52 11.18C0.31 10.76 0.15 10.34 0.09 9.96C0.03 9.59 0.06 9.15 0.38 8.85L0.39 8.83L0.41 8.82L3.03 6.80L3.06 6.78L3.09 6.76C3.37 6.62 3.69 6.62 3.99 6.68C4.21 6.72 4.45 6.80 4.68 6.90L11.57 0.32L11.57 0.31L11.58 0.31C12.02 -0.08 12.64 -0.01 13.15 0.20ZM12.11 0.97L5.13 7.64C5.67 7.99 6.24 8.50 6.76 9.13C7.29 9.76 7.70 10.42 7.98 11.02L14.95 4.36C15.03 4.28 15.12 4.05 15.00 3.57C14.88 3.12 14.60 2.57 14.16 2.06C13.73 1.54 13.25 1.18 12.85 1.01C12.43 0.83 12.20 0.89 12.11 0.97ZM0.92 9.51L1.32 9.19C2.10 9.05 3.44 9.85 4.58 11.21C5.67 12.51 6.27 13.93 6.14 14.76L5.95 14.98C5.88 15.03 5.78 15.08 5.60 15.07C5.41 15.07 5.16 15.00 4.86 14.86C4.28 14.58 3.58 14.03 2.92 13.25C2.25 12.46 1.62 11.55 1.24 10.78C1.05 10.39 0.94 10.06 0.90 9.82C0.87 9.61 0.90 9.53 0.92 9.51ZM7.04 6.35L9.37 4.12C9.53 3.97 9.78 3.98 9.92 4.16C10.07 4.33 10.05 4.59 9.89 4.74L7.57 6.96C7.40 7.12 7.16 7.10 7.02 6.93C6.87 6.76 6.88 6.50 7.04 6.35Z"})}),"FlashlightIcon"),follow:ew(e=>(0,C.tZ)("svg",{width:"16",height:"16",viewBox:"0 0 25 26",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e,children:(0,C.tZ)("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M12.73 0.59C10.36 0.58 8.04 1.30 6.06 2.65C4.09 3.99\n        2.55 5.91 1.64 8.15C0.73 10.39 0.49 12.85 0.95 15.23C1.41\n        17.61 2.56 19.80 4.23 21.51C5.91 23.23 8.05 24.40 10.37\n        24.87C12.70 25.34 15.11 25.10 17.30 24.17C19.50 23.25 21.37\n        21.68 22.69 19.66C24.01 17.64 24.71 15.27 24.71 12.85C24.71\n        9.60 23.45 6.48 21.20 4.18C18.96 1.88 15.91 0.59 12.73\n        0.59ZM6.46 16.68C5.72 16.68 4.99 16.46 4.38 16.04C3.76\n        15.61 3.28 15.01 2.99 14.31C2.71 13.61 2.64 12.84 2.78 12.10C2.93 11.35 3.29 10.67 3.81 10.13C4.34\n        9.60 5.00 9.23 5.73 9.08C6.46 8.93 7.22 9.01 7.90 9.30C8.59 9.59 9.17 10.08 9.59 10.71C10.00 11.35\n        10.22 12.09 10.22 12.85C10.22 13.35 10.12 13.85 9.93 14.32C9.74 14.78 9.47 15.21 9.12 15.56C8.77\n        15.92 8.35 16.20 7.90 16.39C7.44 16.59 6.95 16.68 6.46 16.68ZM19.44 16.66L18.16 15.36L19.72\n        13.77H12.86V11.92H19.67L18.16 10.39L19.43 9.09L23.14 12.87L19.44 16.66Z"})}),"FollowIcon"),hover:ew(e=>(0,C.tZ)("svg",{width:"16",height:"16",viewBox:"0 0 25 26",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e,children:(0,C.tZ)("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M12.80 0.59C10.43 0.58 8.11 1.30 6.13 2.65C4.16 3.99 2.62 5.91 1.71 8.15C0.80 10.39 0.56 12.85 1.02 15.23C1.49 17.61 2.63 19.80 4.30 21.51C5.98 23.23 8.12 24.40 10.45 24.87C12.77 25.34 15.18 25.10 17.38 24.17C19.57 23.25 21.44 21.68 22.76 19.66C24.08 17.64 24.78 15.27 24.78 12.85C24.78 9.60 23.52 6.48 21.27 4.18C19.03 1.88 15.98 0.59 12.80 0.59ZM12.72 4.22C13.46 4.22 14.19 4.45 14.80 4.87C15.42 5.29 15.90 5.89 16.19 6.59C16.47 7.29 16.55 8.07 16.40 8.81C16.26 9.56 15.90 10.24 15.38 10.78C14.85 11.32 14.18 11.68 13.45 11.83C12.72 11.98 11.97 11.90 11.28 11.61C10.59 11.32 10.01 10.83 9.59 10.20C9.18 9.57 8.96 8.82 8.96 8.06C8.96 7.05 9.36 6.07 10.06 5.35C10.76 4.63 11.72 4.22 12.72 4.22ZM12.76 18.97C8.15 18.97 4.41 18.54 4.41 18.02C4.41 17.49 8.15 17.07 12.76 17.07C17.38 17.07 21.11 17.49 21.11 18.02C21.11 18.55 17.38 18.97 12.76 18.97Z"})}),"HoverIcon"),orbit:ew(e=>(0,C.tZ)("svg",{width:"16",height:"16",viewBox:"0 0 25 26",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e,children:(0,C.tZ)("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M12.69 0.59C10.32 0.58 8.00 1.30 6.03 2.65C4.05 3.99 2.51 5.91 1.60 8.15C0.70 10.39 0.46 12.85 0.92 15.23C1.38 17.61 2.52 19.80 4.20 21.51C5.87 23.23 8.01 24.40 10.34 24.87C12.67 25.34 15.08 25.10 17.27 24.17C19.46 23.25 21.34 21.68 22.65 19.66C23.97 17.64 24.68 15.27 24.68 12.85C24.68 9.60 23.41 6.48 21.17 4.18C18.92 1.88 15.87 0.59 12.69 0.59ZM12.69 20.66C10.66 20.66 8.71 19.84 7.26 18.39C5.81 16.93 4.98 14.95 4.94 12.88L3.41 14.43L2.28 13.27L5.59 9.87L8.91 13.27L7.78 14.43L6.42 13.04C6.49 14.36 6.94 15.62 7.74 16.67C8.53 17.71 9.61 18.47 10.84 18.86C12.08 19.25 13.40 19.24 14.62 18.84C15.85 18.44 16.93 17.66 17.71 16.60L18.76 17.68C18.03 18.60 17.11 19.35 16.06 19.87C15.01 20.39 13.86 20.66 12.70 20.66L12.69 20.66ZM10.10 12.74C10.10 12.23 10.24 11.73 10.52 11.30C10.80 10.87 11.20 10.54 11.66 10.35C12.13 10.15 12.64 10.10 13.13 10.20C13.62 10.30 14.07 10.55 14.43 10.91C14.78 11.27 15.03 11.73 15.12 12.24C15.22 12.74 15.17 13.26 14.98 13.74C14.79 14.21 14.46 14.62 14.04 14.90C13.63 15.19 13.14 15.34 12.63 15.34C11.96 15.34 11.32 15.06 10.84 14.58C10.36 14.09 10.10 13.43 10.10 12.74ZM19.70 15.76L16.39 12.37L17.51 11.22L18.96 12.69C18.94 11.34 18.51 10.02 17.72 8.93C16.93 7.84 15.82 7.03 14.56 6.62C13.3 6.22 11.94 6.23 10.68 6.65C9.42 7.08 8.33 7.90 7.55 9.01L6.50 7.93C7.20 6.93 8.13 6.13 9.21 5.58C10.29 5.03 11.48 4.76 12.69 4.80C14.73 4.80 16.70 5.63 18.15 7.10C19.60 8.58 20.43 10.58 20.44 12.67L21.88 11.21L23.01 12.36L19.70 15.76Z"})}),"OrbitIcon"),reveal:ew(e=>(0,C.tZ)("svg",{width:"16",height:"16",viewBox:"0 0 25 26",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e,children:(0,C.tZ)("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M12.76 0.59C10.39 0.58 8.07 1.30 6.10 2.65C4.12 3.99 2.59 5.91 1.68 8.15C0.77 10.39 0.53 12.85 0.99 15.23C1.45 17.61 2.59 19.80 4.27 21.51C5.95 23.23 8.08 24.40 10.41 24.87C12.74 25.34 15.15 25.10 17.34 24.17C19.53 23.25 21.41 21.68 22.73 19.66C24.04 17.64 24.75 15.27 24.75 12.85C24.75 9.60 23.49 6.48 21.24 4.18C18.99 1.88 15.94 0.59 12.76 0.59ZM9.80 4.58C10.30 4.58 10.79 4.74 11.21 5.02C11.63 5.31 11.95 5.71 12.14 6.19C12.34 6.66 12.39 7.18 12.29 7.68C12.19 8.19 11.95 8.65 11.59 9.01C11.24 9.38 10.79 9.62 10.29 9.72C9.80 9.82 9.29 9.77 8.83 9.58C8.36 9.38 7.97 9.05 7.69 8.62C7.41 8.19 7.26 7.69 7.26 7.18C7.26 6.49 7.53 5.83 8.00 5.34C8.48 4.86 9.12 4.58 9.80 4.58ZM3.87 17.95L7.78 11.86L10.71 16.38L15.60 8.02L21.74 17.95H3.87Z"})}),"RevealIcon"),rewards:ew(e=>(0,C.tZ)("svg",{width:"16",height:"16",viewBox:"0 0 16 16",...e,xmlns:"http://www.w3.org/2000/svg",children:(0,C.tZ)("path",{d:"M7.20 16H8.33V13.87C10.81 13.70 12.28 12.13 12.28 10.16C12.28 7.88 10.83 7.03 8.33 6.63V3.17C9.35 3.34 9.88 3.93 10.06 4.95H12.06C11.86 2.86 10.50 1.73 8.33 1.53V0H7.20V1.51C4.90 1.67 3.40 3.07 3.40 4.94C3.40 7.08 4.70 7.99 7.20 8.42V12.21C5.59 12.01 5.22 11.08 5.09 9.99H3C3.16 12.03 4.31 13.67 7.20 13.87V16V16ZM5.39 4.75C5.39 3.91 6.03 3.25 7.19 3.12V6.44C5.71 6.14 5.39 5.63 5.39 4.75ZM10.20 10.36C10.20 11.35 9.46 12.06 8.33 12.21V8.60C9.80 8.92 10.20 9.39 10.20 10.36V10.36Z"})}),"RewardsIcon"),waffle:ew(e=>(0,C.tZ)("svg",{width:"24",height:"24",viewBox:"0 0 24 24",fill:"none",...e,xmlns:"http://www.w3.org/2000/svg",children:(0,C.tZ)("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M3.98 6C3.98 4.88 4.88 3.98 5.99 3.98C7.11 3.98 8.01\n        4.88 8.01 6C8.01 7.11 7.11 8.01 5.99 8.01C4.88 8.01 3.98\n        7.11 3.98 6ZM3.98 12C3.98 10.88 4.88 9.98 5.99 9.98C7.11\n        9.98 8.01 10.88 8.01 12C8.01 13.11 7.11 14.01 5.99\n        14.01C4.88 14.01 3.98 13.11 3.98 12ZM5.99 15.98C4.88\n        15.98 3.98 16.88 3.98 18C3.98 19.11 4.88 20.01 5.99 20.01C7.11\n         20.01 8.01 19.11 8.01 18C8.01 16.88 7.11 15.98 5.99 15.98ZM9.98\n         6C9.98 4.88 10.88 3.98 11.99 3.98C13.11 3.98 14.01 4.88 14.01\n         6C14.01 7.11 13.11 8.01 11.99 8.01C10.88 8.01 9.98 7.11 9.98\n         6ZM11.99 9.98C10.88 9.98 9.98 10.88 9.98 12C9.98 13.11 10.88\n         14.01 11.99 14.01C13.11 14.01 14.01 13.11 14.01 12C14.01\n         10.88 13.11 9.98 11.99 9.98ZM9.98 18C9.98 16.88 10.88 15.98\n         11.99 15.98C13.11 15.98 14.01 16.88 14.01 18C14.01 19.11 13.11\n         20.01 11.99 20.01C10.88 20.01 9.98 19.11 9.98 18ZM18.00 3.98C16.88\n         3.98 15.98 4.88 15.98 6C15.98 7.11 16.88 8.01 18.00 8.01C19.11 8.01\n         20.01 7.11 20.01 6C20.01 4.88 19.11 3.98 18.00 3.98ZM15.98 12C15.98\n         10.88 16.88 9.98 18.00 9.98C19.11 9.98 20.01 10.88 20.01 12C20.01\n         13.11 19.11 14.01 18.00 14.01C16.88 14.01 15.98 13.11 15.98\n         12ZM18.00 15.98C16.88 15.98 15.98 16.88 15.98 18C15.98 19.11\n         16.88 20.01 18.00 20.01C19.11 20.01 20.01 19.11 20.01 18C20.01\n         16.88 19.11 15.98 18.00 15.98Z"})}),"WaffleIcon")},lw=Object.keys(lk),lx=ew(({iconName:e,className:t,fill:n,...o})=>{let r=lk[e];return r?(0,C.tZ)(r,{className:(0,E.cx)(`icon-${M(e)}`,t),fill:n??sc("--icon-color"),...o}):null},"CustomIcon"),lS={"arrow-left":j.Z,"arrow-right":R.Z,"align-bottom":L.Z,"bars-triangle":A.Z,"chat-outline":V.Z,"chevron-down":q.Z,"chevron-left":Z.Z,"chevron-right":B.Z,"chevron-up":Q.Z,"external-link":H.Z,"heart-outline":G.Z,list:U.Z,"play-filled":W.Z,"sort-triangles":K.Z,"transition-curve-up":Y.Z,add:X.Z,bars:J.Z,camera:ee.Z,cart:et.Z,check:en.Z,cross:eo.Z,globe:er.Z,remove:ei.Z,rocket:ea.Z,search:es.Z,trophy:el.Z},lC={chevron:q.Z,"chevron-inverted":Q.Z,close:eo.Z,chat:V.Z,"camera-rounded":ee.Z};Object.keys(lS),Object.keys(lC);var lF=ew(({name:e,className:t,fill:n,size:o=16})=>{if(ls("sdsm-icon"),e in lS){let r=lS[e];return(0,C.tZ)(r,{size:o,fill:n,className:(0,E.cx)("sdsm-icon",t)})}if(e in lC){let r=lC[e];return(0,C.tZ)(r,{size:o,fill:n,className:(0,E.cx)("sdsm-icon",t)})}return lw.includes(e)?(0,C.tZ)(lx,{width:o,height:o,fill:n,iconName:e,className:(0,E.cx)("sdsm-icon",t)}):null},"Icon"),l_=ew(e=>["duration","curve","delay","iterationCount","direction","fillMode","playState"].map(t=>e[t]).filter(e=>!!e).join(" "),"animationPropsToCssProp"),lN=ew((e={duration:"250ms",curve:"ease-out"})=>(0,E.css)`
  @keyframes fadeIn {
    0% {
      opacity: 0;
    }
    100% {
      opacity: 1;
    }
  }

  animation: fadeIn ${l_(e)};
`,"opacityFadeInCss"),l$=ew((e={duration:"250ms",curve:"ease-out"})=>(0,E.css)`
  @keyframes fadeOut {
    0% {
      opacity: 1;
    }
    100% {
      opacity: 0;
    }
  }

  animation: fadeOut ${l_(e)};
`,"opacityFadeOutCss"),lE="cubic-bezier(0.4, 0.0, 0.2, 1)",lO="cubic-bezier(0.0, 0.0, 0.2, 1)",lD=ew((e={duration:"250ms",curve:lE})=>(0,E.css)`
  @keyframes slideUp {
    0% {
      transform: translateY(100vh);
    }
    100% {
      transform: translateY(0);
    }
  }

  animation: slideUp ${l_(e)};
`,"slideUpCss"),lT=ew((e={duration:"250ms",curve:lO})=>(0,E.css)`
  @keyframes slideDown {
    0% {
      transform: translateY(0);
    }
    100% {
      transform: translateY(100vh);
    }
  }

  animation: slideDown ${l_(e)};
`,"slideDownCss"),lz=(0,E.css)`
  & > summary {
    /* Prevents selection when rapidly expanding and collapsing the content */
    user-select: none;

    cursor: pointer;
  }

  scroll-margin: 40px;

  /* Prevents keyboard selection of focusable elements that the panel is collapsed. */
  &[data-state='off'] > summary + * {
    visibility: hidden;
  }
`,lM=ew((e=250)=>(0,E.css)`
  &[data-state='turning-on'] > summary + * {
    ${lN({duration:`${e}ms`,curve:"ease-out"})}
  }

  &[data-state='turning-off'] > summary + * {
    ${l$({duration:`${e}ms`,curve:"ease-out"})}
  }
`,"detailsAnimationCssFn"),lI=(0,E.css)`
  justify-self: center;
  width: 12px;
  height: 12px;

  transform: rotate(0deg); /* 'up' */
  transition: transform 250ms;

  details[data-state='on'] > summary > &,
  details[data-state='turning-on'] > summary > & {
    transform: rotate(180deg); /* 'down', ccw */
  }

  *[dir='rtl'] details[data-state='on'] > summary > &,
  *[dir='rtl'] details[data-state='turning-on'] > summary > & {
    transform: rotate(-180deg); /* 'down', clockwise */
  }
  position: absolute;
  right: 0;

  /* stylelint-disable-next-line no-descending-specificity */
  *[dir='rtl'] & {
    right: unset;
    left: 0;
  }

  /*
   * Note: 6px is half of default height. Overwrite this via 'summary > svg.icon-chevron {...}'
   * TODO: Make the height of the chevron icon a CSS var.
   */
  top: calc(50% - 6px);
`,lP=(0,E.css)`
  margin: 0;
  position: relative; /* To position the chveron absolutely. */

  /* In webkit the arrow is rendered as a list icon; so we hide that, too. */
  /* We add our own arrow so we need to hid the default one. */
  ::-webkit-details-marker {
    display: none;
  }
  list-style-type: none;
`,lj=ew(({showChevron:e=!0,chevronProps:t,onToggle:n,summary:o,summaryProps:r,children:i,className:a,fadeInAnimation:s=!0,transitionDurationMs:l,open:c,detailsRef:u,summaryRef:d,disableScrollToOnOpen:h,...p})=>{let f=!z(c);ls("sdsm-detail-summary");let{state:m,toggle:g}=ly({transitionDurationMs:l});(0,b.useEffect)(()=>{f&&g(c?"on":"off")},[f,c,g]),(0,b.useEffect)(()=>{!f&&n&&n("on"===m?"on":"off")},[n,m,f]);let v=ew(e=>{e.preventDefault(),f?n?.(c?"off":"on"):g()},"toggleWithAnimation"),y=!!f||"off"!==m,k=ew(e=>{if(f?c:e.currentTarget.open){e.persist();let t=e.currentTarget;requestAnimationFrame(()=>t.scrollIntoView({behavior:"smooth",block:"nearest"}))}},"scrollIntoViewIfOpen"),{className:w,dataset:x,...S}=r??{},F=ew(e=>{if("string"==typeof e)return e},"getTitle");return(0,C.BX)("details",{ref:u,className:(0,E.cx)("sdsm-detail-summary",lz,a,s?lM(l):void 0),onToggle:h?void 0:k,open:y,"data-state":m,...p,children:[(0,C.BX)("summary",{ref:d,role:"button",tabIndex:0,title:F(o),className:(0,E.cx)(w,lP),onClick:v,...s6(x),...S,children:[o,e&&(0,C.tZ)(lF,{className:(0,E.cx)(lI,t?.className),name:"chevron-down",fill:t?.fill})]}),i]})},"DetailsSummary");lj.displayName="DetailsSummary";var lR=(0,E.css)`
  &.small {
    --size: 28px;
    --icon-size: 24px;
  }
  &.medium {
    --size: 32px;
    --icon-size: 24px;
  }
  &.large {
    --size: 40px;
    --icon-size: 24px;
  }
  &.extra_large {
    --size: 64px;
    --icon-size: 40px;
  }

  /* These are meant to be easily overrideable via className prop. */
  width: var(--size);
  height: var(--size);

  cursor: pointer;
  transition: transform 0.2s linear;

  border-radius: 50%;
  box-shadow: ${sc("--box-shadow-xs")};

  /* Center */
  display: flex;
  align-items: center;
  justify-content: center;

  --icon-color: ${sc("--icon-button-fg-color")};
  background: ${sc("--icon-button-bg-color")};
  border: ${sc("--icon-button-border-width")} solid ${sc("--icon-button-border-color")};

  :hover {
    background: ${sc("--icon-button-hover-bg-color")};
    border: ${sc("--icon-button-border-width")} solid ${sc("--icon-button-hover-border-color")};
    transform: translate(0, -1px);
  }

  :active,
  .active {
    transform: translate(0, 1px);
  }

  :disabled,
  .disabled {
    cursor: not-allowed;
    transform: none;
    --icon-color: ${sc("--icon-button-disabled-fg-color")};
    background: ${sc("--icon-button-disabled-bg-color")};
    border: ${sc("--icon-button-border-width")} solid ${sc("--icon-button-disabled-border-color")};
  }

  > .icon {
    width: var(--icon-size);
    height: var(--icon-size);
  }
`,lL=(0,b.forwardRef)((e,t)=>{let{url:n,iconName:o,className:r,iconClassName:i,onClick:a,disabled:s,size:l="medium",...c}=e;ls("sdsm-icon-button");let u=(0,E.cx)("sdsm-icon-button",lR,l,{disabled:s},r),d=(0,C.tZ)(lF,{name:o,size:24,fill:sc("--icon-color"),className:(0,E.cx)("icon",i)}),h=(0,b.useCallback)(()=>{s||a?.()},[s,a]);return n?(0,C.tZ)("a",{href:n,ref:t,className:u,onClick:h,...c,children:d}):(0,C.tZ)("button",{ref:t,disabled:s,className:u,onClick:h,...c,children:d})});function lA(){let{getLowEntropyHints:e,getCachedHighEntropyHints:t}=(0,b.useContext)(a4),n=e().isMobile,o=t()?.viewportWidth,[r,i]=(0,b.useState)((o?o<=768:n)?ss.Mobile:ss.Desktop),{width:a}=lt();return(0,b.useEffect)(()=>{i((a??Number.POSITIVE_INFINITY)<=768?ss.Mobile:ss.Desktop)},[a]),r}ew(lA,"useMediaMode");var lV="min(8vw, 112px)",lq="global-links",lZ="local-nav",lB="max(-8vw, -112px)",lQ="level-tracker",lH="--next-level-line-height",lG={headerTopLevel:"sdsm-global-header",openButton:"sdsm-global-header-open",navItems:"sdsm-global-header-local-nav-items",endContents:"sdsm-global-header-end-contents"};function lU(e){return function(t){if(!e)return!1;try{let n=`${e.protocol}//${e.hostname}`,o=new URL(t,n),r=e.hostname,i=e.pathname;return r===o.hostname&&i===o.pathname}catch(e){return!1}}}function lW(e,t=lU("undefined"==typeof window?void 0:window?.location)){let n=new Set;return e.map(e=>lK(n,e,t)),n}ew(lU,"defaultIsUrlCurrent"),ew(lW,"getNavigationBreadcrumbs");var lK=ew((e,t,n,o=[])=>{if(!t.subItems?.length){t.url&&n?.(t.url)&&(e.add(t.id),o.map(t=>e.add(t.id)));return}for(let r of t.subItems)lK(e,r,n,[...o,t])},"traverseTree"),lY=(0,b.createContext)({}),lX=ew((e,t)=>(0,E.css)`
  & > * {
    margin-left: ${t/2}px;
    margin-right: ${t/2}px;
  }

  & > *:first-child {
    margin-left: ${e}px;
    margin-right: ${t/2}px;

    *[dir='rtl'] & {
      margin-right: ${e}px;
      margin-left: ${t/2}px;
    }
  }

  /* stylelint-disable-next-line no-descending-specificity */
  & > *:last-child {
    margin-left: ${t/2}px;
    margin-right: ${e}px;

    *[dir='rtl'] & {
      margin-right: ${t/2}px;
      margin-left: ${e}px;
    }
  }
`,"childrenMarginCss"),lJ=(0,E.css)`
  height: ${64}px;
  max-height: ${64}px;
  display: flex;
  align-items: center;
  /* TODO: this doesnt really show well, and needs to work with subnav/banners. */
  /* box-shadow: ${sc("--box-shadow-m")}; */

  /*
   * NOTE: These are referring to content, because the header color actually translates to the
   * Navigation screen that opens when the nav is expanded.
   */
  background-color: ${le("--content-bg")};
  color: ${le("--content-color")};

  /* Prevent the header from expanding the body to fit its contents. */
  max-width: 100vw;
  overflow-x: clip;
  overflow-y: revert;

  /* Stretch the header to the sides and ensure it's visible. */
  position: sticky;
  top: 0;
  left: 0;
  right: 0;
  z-index: ${100};

  opacity: 1;
  &.hidden {
    opacity: 0;
    transform: translateY(-100%);
  }
  transition: opacity 250ms ease-in, transform 250ms ease-out;

  ${lX(20,16)}
`,l0=(0,E.css)`
  /* Prevent the header from expanding the body to fit its contents. */
  max-width: 100vw;
  overflow-x: clip;

  /* Stretch the header to the sides and ensure it's visible. */
  position: sticky;
  top: 0;
  left: 0;
  right: 0;
  z-index: ${100};

  opacity: 1;
  &.hidden {
    opacity: 0;
    transform: translateY(-100%);
  }
  transition: opacity 250ms ease-in, transform 250ms ease-out;
`,l1=(0,E.css)`
  ${lJ}
  & > *:first-child,
  & > *:last-child {
    display: flex;
    flex-grow: 1;
    flex-basis: 100px;
  }

  & > *:first-child {
    justify-content: flex-start;
  }

  & > *:last-child {
    justify-content: flex-end;
  }
`,l2=(0,E.css)`
  > svg {
    fill: ${sc("--global-header-item-color")};
    transition: all 0.15s ease-out;
  }

  &&& {
    background: transparent;
    border: 2px solid ${sc("--global-header-item-color")};
    padding: 0 7px; /* Must override the padding on button. */
  }

  &&&:hover {
    border-color: ${sc("--global-header-item-hover-color")};

    /* Clearing out styles from the regular button. */
    box-shadow: none;
    transform: none;

    > svg {
      fill: ${sc("--global-header-item-hover-color")};
    }
  }
`,l3=(0,E.css)`
  background-color: transparent;
  border: none;
  color: ${le("--content-color")};
  padding: 4px 8px;
  cursor: pointer;
`,l5=(0,E.css)`
  display: flex;
  align-items: center;

  ${lX(0,16)}
`,l4=(0,E.css)`
  box-shadow: none;
  /* Hack to hide icons in CTAs. */
  i[class^='icon-'],
  img,
  svg {
    display: none;
  }
`,l6=(0,E.css)`
  & > *:not(:first-child) {
    margin-left: 16px;
    *[dir='rtl'] & {
      margin-left: unset;
      margin-right: 16px;
    }
  }

  /* NOTE: Triple specific selector to override button's double-specific styles. */
  & .${"sdsm-button"}.${"sdsm-button"}.${"sdsm-button"} {
    ${l4}
    font-size: smaller;
    margin-right: 0;
    padding: 4px;
    *[dir='rtl'] & {
      margin-right: unset;
      margin-left: 0;
    }
  }
`;(0,E.css)`
  width: 40px;
  height: 40px;
  border: 1px solid ${"#C7C7CC"};
  border-radius: 50%;
  background-color: transparent;
`;var l8=(0,E.css)`
  ${lJ}

  padding: 0 40px;

  /**
   * We are overriding the overflow from globalHeaderCss because of a bug
   * on Safari 16.2. Not sure why this works so this is a bandaid. Ideally we don't do this
   */
  overflow: visible;

  & > nav {
    flex-grow: 1;
    display: flex;
    justify-content: flex-end;
  }

  span {
    font-weight: ${sc("--action-desktop-font-weight")};
    font-size: ${sc("--action-desktop-font-size")};
    margin: 0 ${sc("--spacing-xs")};

    ${st} {
      font-weight: ${sc("--action-mobile-font-weight")};
      font-size: ${sc("--action-mobile-font-size")};
    }
  }
`,l9=(0,E.css)`
  margin: -7.5px -5px;
`,l7=ew(({logo:e,siteName:t,className:n,cta:o,endChildrenClassName:r,localNavDesktop:i,toggleExpanded:a,showNavScreen:s,showGlobalLinks:l=!0,dataset:c})=>(0,C.tZ)("header",{className:(0,E.cx)(l0,n),...s6(c),children:(0,C.BX)("div",{className:l8,children:[l&&s&&(0,C.tZ)(lL,{size:"large",className:l2,iconName:"waffle",onClick:a,"data-testid":lG.openButton}),e,(0,C.tZ)("span",{children:t}),(0,C.tZ)("nav",{"data-testid":lG.navItems,children:i}),(0,C.tZ)("aside",{className:r??l5,"data-testid":lG.endContents,children:o})]})}),"GlobalHeaderDesktop");l7.displayName="GlobalHeaderDesktop";var ce={navTopLevel:"sdsm-global-nav-screen",closeButton:"sdsm-global-nav-screen-close",navItems:"sdsm-global-nav-screen-items",navHighlights:"sdsm-global-nav-screen-highlights"},ct=ew(({logo:e,cta:t,className:n,isExpanded:o,toggleExpanded:r,showNavScreen:i,endChildrenClassName:a,dataset:s})=>(0,C.BX)("header",{className:(0,E.cx)(l1,n),...s6(s),children:[i&&(0,C.tZ)("button",{onClick:r,className:l3,"data-testid":o?ce.closeButton:lG.openButton,children:o?(0,C.tZ)(lF,{name:"cross",size:23,fill:le("--content-color")}):(0,C.tZ)(lF,{name:"bars",className:l9,size:30,fill:le("--content-color")})}),e,(0,C.tZ)("aside",{className:a??l6,"data-testid":lG.endContents,children:t})]}),"GlobalHeaderMobile");ct.displayName="GlobalHeaderMobile";var cn=ew(({backgroundColor:e,className:t,defaultGroupKey:n,children:o,displayed:r,onToggleExpanded:i,stayOpenInvariant:a,trackingSiteName:s,showNavScreen:l=!0,dataset:c,isUrlCurrent:u,headerNavigationTree:d,...h})=>{ls("sdsm-header");let p=lA(),f=(0,b.useMemo)(()=>lW(d??[],u),[d,u]),{state:m,toggle:g}=ly({transitionDurationMs:450,onToggle:(e,t)=>{i&&("turning-on"===e&&i(!0),"off"===e&&i(!1));let n=document?.activeElement;n?.blur()}});(0,b.useEffect)(()=>g("off"),[p,a]);let[v,y]=(0,b.useState)(p===ss.Mobile?"home":n??""),k=(0,E.cx)("sdsm-header",e?s7(e):null,r??!0?null:"hidden",t);c?.testid||(c={...c,testid:lG.headerTopLevel});let w="off"!==m,x=p===ss.Mobile?ct:l7;return(0,C.BX)(lY.Provider,{value:{mode:p,groupKey:v,setGroupKey:y,toggleExpanded:g,screenState:m,trackingSiteName:s,navigationBreadcrumbs:f},children:[(0,C.tZ)(x,{...h,isExpanded:w,toggleExpanded:g,className:k,showNavScreen:l,dataset:c}),w&&l&&o]})},"GlobalHeader");cn.displayName="GlobalHeader";var co=(0,E.css)`
  display: flex;
  justify-content: space-between;

  & > .icon-external-link {
    color: ${sc("--global-header-item-color")};
    font-size: 16px;
    vertical-align: middle;
    align-self: center;
  }

  & > a {
    position: relative;
  }

  & a.selected {
    color: ${sc("--global-header-item-active-color")};
  }

  padding: 0;
`,cr=(0,E.css)`
  summary.selected {
    color: ${sc("--global-header-item-active-color")};
  }
`,ci=(0,E.css)`
  &.icon-external-link {
    color: ${sc("--global-header-item-color")};
    font-size: 14px;
    margin: 0 9px;
  }
`,ca=(0,E.css)`
  width: 20px;
  height: 20px;
  fill: ${sc("--global-header-item-color")};

  summary.selected > & {
    fill: ${sc("--global-header-item-active-color")};
  }
`,cs="--children-count",cl=(0,E.css)`
  overflow: hidden;

  height: 0;
  opacity: 0;

  transition-property: opacity, height;
  transition-duration: ${300}ms;
  transition-timing-function: ease-in;

  details[data-state='on'] &,
  details[data-state='turning-on'] & {
    height: calc(var(${lH}) * var(${cs}));
    opacity: 1;

    transition-timing-function: ${lO};
  }
`,cc=(0,E.css)`
  list-style-type: none;
  padding: 0 0 0 16px;
  margin: 0;

  *[dir='rtl'] & {
    padding-left: initial;
    padding-right: 12px;
  }

  ${sn} {
    ${cl}
  }
`,cu=(0,E.css)`
  transition: color 0.125s;
  white-space: nowrap;
  scroll-margin: ${lV};

  /* Hide the text that doesn't fit. */
  text-overflow: ellipsis;
  overflow: hidden;

  &:hover,
  &:active,
  &:focus-within,
  details[data-state='on'] > & {
    color: ${sc("--global-header-item-hover-color")};
  }

  /** Displays the knob on active and hover in desktop mode only */
  ${sn} {
    /* Move the summary over so we can position the knob inside */
    left: ${lB};
    padding-left: ${lV};
    width: calc(20px + 100%);

    *[dir='rtl'] & {
      left: unset;
      padding-left: unset;
      right: ${lB};
      padding-right: ${lV};
    }

    &::before {
      content: ' ';
      background-color: ${sc("--global-header-item-hover-color")};
      position: absolute;
      left: 0;
      top: 10px;
      width: 7px;
      height: 32px;
      border-radius: ${sc("--border-radius-xs")};
      visibility: hidden;
    }

    *[dir='rtl'] &::before {
      left: unset;
      right: 0;
    }

    &:focus-within::before,
    &:hover::before,
    &:active::before,
    details[data-state='on'] > &::before {
      visibility: visible;
    }
  }
`,cd=(0,E.css)`
  position: relative;
`,ch=ew(({navGroupKey:e,title:t,mobileHighlight:n,children:o})=>{let{groupKey:r,setGroupKey:i,mode:a,screenState:s}=(0,b.useContext)(lY),[l,c]=(0,b.useState)(!1);(0,b.useEffect)(()=>{let t;let n=r===e;if(l!==n)return n&&"on"===s?t=setTimeout(c.bind(null,n),300):c(n),()=>{t&&clearTimeout(t)}},[r,e]);let u=(0,C.tZ)("ol",{className:cc,role:"menu",children:b.Children.map(o,(e,t)=>(0,C.tZ)("li",{role:"menuitem",children:e},`list-item-${t}`))}),d=ew(t=>{"on"===t&&i&&i(e)},"setGroupKeyOnToggleOn"),h=a===ss.Mobile,p=(0,E.css)`
    ${cs}: ${b.Children.count(o)};
  `;return(0,C.BX)(lj,{showChevron:a===ss.Mobile,onToggle:d,open:h?void 0:l,summary:t,summaryProps:{className:cu},chevronProps:{className:ca},className:(0,E.cx)(lQ,h?void 0:cd,p),fadeInAnimation:h,transitionDurationMs:300,"data-testid":`group-${e}`,children:[u,h&&n]})},"GlobalNavGroup");ch.displayName="GlobalNavGroup";var cp="highlight-text",cf="highlight-text-title",cm="highlight-text-body",cg="highlight-media-mobile",cv="highlight-media-desktop",cb=ew(e=>(0,E.css)`
  box-sizing: border-box;

  /* Make the image/video cover the media space. These selectors are deep to account for any wrapper elements. */
  & > .${e} * {
    width: 100%;
    height: 100%;
  }
  & > .${e} img,
  & > .${e} video {
    object-fit: cover;
    border-radius: ${sc("--border-radius-l")};
  }

  & > .${cg} img {
    width: 90%;
    margin: 0;
  }
`,"highlightMediaCss"),cy=(0,E.css)`
  position: relative; /* For positioning the card */
  filter: drop-shadow(0px 0px 32px rgba(0, 0, 0, 0.12)); /* Faint highlight shadow */

  & > .${cv} {
    margin: 0;
    width: 100%;
    height: 100%;
  }

  ${cb(cv)}
`,ck=ew(e=>(0,E.css)`
  /* stylelint-disable-next-line */
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: ${e};
`,"lineClampCss"),cw=(0,E.css)`
  box-sizing: border-box;
  color: ${sc("--foreground-color")};
  background-color: ${sc("--background-color")};

  padding: 0;
  border-radius: ${sc("--border-radius-m")};
  box-shadow: ${sc("--box-shadow-m")};
  border: 1px solid rgb(233, 234, 235);

  & > .${cp} {
    & > .${cf} {
      color: ${"#121314"};

      font-weight: 500;
      font-size: 18px;
      line-height: 20px;
      white-space: nowrap;

      display: inline-block;

      max-width: 100%;

      overflow: hidden;
      text-overflow: ellipsis;

      /* The line height here is greater than in body, so we move it slightly so the top margin
         looks proportional to the card border. */
      position: relative;
      bottom: 1px;
    }

    & > .${cm} {
      /* Default <p> sets margin to 1em which is larger than what we want. */
      margin-block-start: 2px;
      margin-block-end: 2px;

      font-size: 12px;
      line-height: 16px;
      font-weight: 400;

      color: ${"#53575B"};

      ${ck(3)}
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }
`,cx=(0,E.css)`
  width: 35%;
  min-width: 500px;
  padding: 0 16px;

  *[dir='rtl'] & {
    right: unset;
    left: ${65}px;
  }

  .${"cta"} {
    margin-left: 0;
  }

  *[dir='rtl'] & .${"cta"} {
    margin-left: ${16}px;
    margin-right: 0;
  }
`,cS=(0,E.css)`
  width: calc(100% - ${130}px);

  &,
  & > .${"cta"}, & > .${"cta"} .${"sdsm-button"} {
    display: flex;
    flex-direction: column;
  }

  & > .${"cta"} {
    margin-top: 0;
  }
`,cC=(0,E.css)`
  ${cw}

  display: flex;
  align-items: center;

  & > .${"cta"} {
    flex-grow: 0;
  }

  & > .${cp} {
    flex-grow: 1;
  }

  & > .${cp}, & > .${"cta"} {
    margin: 16px;
  }

  position: absolute;
  right: ${65}px;
  bottom: 30px;

  @media screen and (min-width: ${1040}px) {
    ${cx}
  }

  @media screen and (max-width: ${1040}px) {
    ${cS}
  }
`;(0,E.css)`
  width: 100%;
  cursor: pointer;
  margin-top: 15px;
  margin-bottom: 50px;

  /**
   * NOTE: These values are important!
   * This first value is the height of the highlight.
   * The second is the height of the highlight description. The content is too tall
   * So we clip it after this size (1-line roughly)
   */
  display: grid;
  grid-template-rows: 40vw 20px;

  & > * {
    user-select: none;
  }

  & > .${cp} {
    font-weight: 400;
    font-size: 14px;
    line-height: 40px;
    margin: 0 12px;
  }

  & > .${cg} {
    border-radius: ${sc("--border-radius-m")};
    margin: 0 20px 0 12px;
  }

  [dir='rtl'] & > .${cg} {
    margin: 0 12px 0 20px;
  }

  ${cb(cg)}
`;var cF=(0,E.css)`
  transition: transform ${500}ms ${lE};
  transform: translateY(
    calc(((-100% + ${50}px) * var(--selected-index) + ${30}px))
  );
  height: 100%;

  margin: 0 32px 0 20px;

  *[dir='rtl'] & {
    margin: 0 20px 0 32px;
  }

  & > * {
    margin-top: 0;
    margin-bottom: ${20}px;
    height: calc(100% - ${70}px);
  }
`,c_=ew(({background:e,callToAction:t,cardTitleV2:n,cardBodyV2:o,cardTitle:r,cardBody:i})=>(0,C.BX)("section",{className:cy,children:[(0,C.tZ)("figure",{className:cv,children:e}),(0,C.BX)("article",{className:(0,E.cx)(s7(a9.White),cC),children:[(0,C.BX)("figcaption",{className:cp,children:[(0,C.tZ)("label",{className:cf,children:n??r}),(0,C.tZ)("p",{className:cm,children:o??i})]}),(0,C.tZ)("div",{className:"cta",children:t})]})]}),"GlobalNavHighlightDesktop");c_.displayName="GlobalNavHighlightDesktop";var cN=ew(e=>(0,b.useContext)(lY).mode===ss.Mobile?null:(0,C.tZ)(c_,{...e}),"GlobalNavHighlight");cN.displayName="GlobalNavHighlight";var c$=ew(({className:e,highlights:t})=>{let n=(0,b.useContext)(lY),o=n.mode===ss.Mobile,[r,i]=(0,b.useState)(0),a=Object.fromEntries([...t.entries()].filter(([e,t])=>!!t));if((0,b.useEffect)(()=>{let e=Object.keys(a).indexOf(n.groupKey??"");-1!==e&&e!==r&&i(e)},[n.groupKey,a,r]),o)return null;let s=(0,E.css)`
    --selected-index: ${r};
  `;return(0,C.tZ)("section",{className:(0,E.cx)(s,cF,e),children:b.Children.toArray(Object.values(a))})},"GlobalNavHighlightReel"),cE=RegExp("^(?:[a-z]+:)?//","i"),cO=ew((e,t)=>{if(!e||!cE.test(e))return e;try{let n=new URL(e);return Object.entries(t).forEach(([e,t])=>{n.searchParams.set(e,t)}),n.toString()}catch{return}},"appendTrackingParameters"),cD=ew(e=>{if(!e||!cE.test(e))return!1;try{return new URL(e).hostname!==window?.location.hostname}catch{return console.warn(`Unable to parse URL '${e}'. It is invalid.`),!1}},"isExternal"),cT={utm_source:"global_nav",utm_medium:"global_header",utm_campaign:"universal_navigation"},cz=ew(({title:e,children:t,isSelected:n})=>(0,C.tZ)(lj,{summary:e,summaryProps:{className:(0,E.cx)({selected:n})},chevronProps:{className:ca},className:(0,E.cx)(lQ,cr),children:(0,C.tZ)("ol",{className:cc,children:b.Children.map(t,e=>(0,C.tZ)("li",{children:e}))})}),"GlobalNavItemWithChildren");cz.displayName="GlobalNavItemWithChildren";var cM=ew(({title:e,href:t,showExternalIcon:n,children:o,onClick:r,isSelected:i,dataset:a})=>{let{navigationBreadcrumbs:s,toggleExpanded:l,trackingSiteName:c}=(0,b.useContext)(lY),u=!!t&&!!s?.has(t),d=void 0===i?u:i;if(o&&(!Array.isArray(o)||o.length))return(0,C.tZ)(cz,{title:e,isSelected:i,children:o});let h=cO(t,c?{...cT,utm_source:ec(c)}:cT),p=cD(t),f=ew(e=>{r&&r(e),p||e.ctrlKey||(l&&l(),setTimeout(()=>window.scrollTo(0,0),500))},"onLinkActivate");return(0,C.BX)("div",{className:co,...s6(a),children:[(0,C.tZ)("a",{className:(0,E.cx)(lQ,{selected:d}),href:h,onClick:f,target:p?"_blank":"_self",rel:"noopener",children:e}),n&&p&&(0,C.tZ)(lF,{className:ci,name:"external-link"})]})},"GlobalNavItem");function cI(){let e=Math.max(document.documentElement.clientHeight,window.innerHeight),t=Math.max(document.documentElement.clientWidth,window.innerWidth);document.documentElement.style.setProperty("--client-height",`${e}px`),document.documentElement.style.setProperty("--client-width",`${t}px`)}function cP(){(0,b.useEffect)(()=>{let e=I(cI,50,{trailing:!0});return window.addEventListener("resize",e),cI(),()=>window.removeEventListener("resize",e)},[])}cM.displayName="GlobalNavItem",ew(cI,"handleResize"),ew(cP,"useSetClientSize");var cj=ew(({backgroundColor:e,thumbColor:t,trackWidth:n})=>(0,E.css)`
  /* Firefox. See developer.mozilla.org/en-US/docs/Web/CSS/CSS_Scrollbars */
  scrollbar-width: ${n};
  scrollbar-color: ${t} ${e};

  /* Edge / Chrome / Safari. See css-tricks.com/custom-scrollbars-in-webkit/ */
  ::-webkit-scrollbar {
    width: ${n};
    background-color: ${e};
  }

  ::-webkit-scrollbar-thumb {
    background-color: ${t};
    border-radius: calc(${n} / 2);
    /* Can't control the width, so we add a border to introduce margin. */
    border: 2px solid ${e};
  }

  ::-webkit-scrollbar-track {
    background-color: ${"transparent"};
  }
`,"scrollbarCss"),cR=ew(e=>`screen-state-${e}`,"screenStateClassName"),cL=(0,E.css)`
  background-color: ${sc("--global-header-nav-screen-bg-color")};
  color: ${sc("--global-header-fg-color")};

  box-sizing: border-box;
  position: fixed;
  left: 0;
  top: 0;
  z-index: ${100};

  & header {
    height: ${64}px;
    padding: 0 60px;
    display: flex;
    align-items: center;
    background-image: linear-gradient(
      0deg,
      transparent 0,
      ${sc("--global-header-nav-screen-bg-color")} ${5}px
    );
    margin-bottom: -${5}px;
    z-index: ${101};

    &.sticky {
      position: sticky;
      top: 0;
    }
  }
`,cA=ew(e=>`.${lQ} `.repeat(e),"levelSelector"),cV=(0,E.css)`
  & a {
    color: ${sc("--global-header-item-color")};
    text-decoration: none;
    position: relative;

    &::after {
      left: 0;
      content: ' ';
      position: absolute;
      bottom: 5px;
      height: 0;
      width: 100%;
      border-bottom: 2px ${sc("--global-header-item-hover-color")} solid;

      transition: ease-in-out;
      transition-duration: 100ms;
      opacity: 0;
    }
  }

  & a:hover,
  & a:focus {
    color: ${sc("--global-header-item-hover-color")};

    &::after {
      opacity: 1;
    }
  }

  & ${cA(1)} {
    font-size: 26px;
    line-height: 54px;
    font-weight: 600;
    ${lH}: 36px;
  }

  & ${cA(2)} {
    font-size: 18px;
    line-height: 36px;
    font-weight: 400;
    ${lH}: 16px;
  }

  & ${cA(3)} {
    font-size: 16px;
    font-weight: 400;
    line-height: 40px;
  }
`,cq=(0,E.css)`
  & a {
    color: ${sc("--global-header-item-color")};
    text-decoration: none;
  }
  & a:hover {
    color: ${sc("--global-header-item-hover-color")};
  }

  .${lQ} {
    font-size: ${sc("--h6-desktop-font-size")};
    font-weight: 500;
    line-height: 40px;
  }

  & ${cA(2)}, & ${cA(3)} {
    font-weight: 400;
  }
`,cZ="highlight",cB="left-nav",cQ=(0,E.css)`
  ${cL}

  ${lT({duration:"500ms"})}
  &.${cR("on")},
  &.${cR("turning-on")} {
    ${lD({duration:"500ms"})}
  }

  width: 100vw;
  max-width: 100vw;
  height: 100vh;
  max-height: 100vh;

  box-sizing: border-box;
  overflow: hidden;

  & > .${cB} {
    height: 100%;
    width: ${400}px;
    max-height: 100%;
    overflow-y: scroll;
    overflow-x: hidden;

    ${cV}

    /* Styling scrollbar */
    ${cj({thumbColor:sc("--global-header-item-color"),backgroundColor:sc("--global-header-nav-screen-bg-color"),trackWidth:"10px"})}
  }

  & > .${cB}.${"no-scroll"} {
    ${cj({thumbColor:sc("--global-header-nav-screen-bg-color"),backgroundColor:sc("--global-header-nav-screen-bg-color"),trackWidth:"10px"})}
  }

  & .menu {
    padding: 0px 20px 0px ${lV};
    box-sizing: border-box;
  }

  *[dir='rtl'] & .menu {
    padding: 0px ${lV} 0px 20px;
  }

  & > .${cZ} {
    position: absolute;
    right: 0;
    top: 0;
    height: 100%;
    width: calc(100% - ${400}px);
    z-index: ${102};

    *[dir='rtl'] & {
      right: unset;
      left: 0;
    }
  }

  & .close {
    position: relative;
  }
`,cH=(0,E.css)`
  ${cL}
  ${cq}

  ${lN({duration:"500ms",curve:"ease-out"})}

  &.${cR("turning-off")} {
    ${l$({duration:"500ms",curve:"ease-out"})}
  }

  overflow: hidden scroll;
  display: flex;
  flex-direction: column;

  & > .${lZ} {
    padding: 0 20px 20px 20px;
    scroll-margin-top: 5px;
  }

  & > .${lq} {
    background-color: ${sc("--global-header-bg-color")};
    color: ${sc("--global-header-fg-color")};
    padding: 20px 20px 40px 20px;
    margin-bottom: -20px; /* Pushing background under the gradient in .highlight. */
    flex-grow: 1; /* Pushes the gray background to the bottom for long pages. */
  }

  & hr {
    border-style: inset;
    border-width: 1px;
    margin-block-start: ${16}px;
    margin-block-end: ${16}px;
  }

  /* On mobile we show the header regardless, so we display the nav below it. */
  /* NOTE: Reducing so that at 75% zoom a sliver of background doesn't show. */
  top: ${63.25}px;
  /* On mobile we need to subtract the header size. */
  /* NOTE: We use client-height instead of 100vh for legacy safari support.
     We should revisit this later. */
  height: calc(var(--client-height) - ${64}px);
  max-height: calc(var(--client-height) - ${64}px);
  width: 100vw;
  max-width: 100vw;
`,cG=(0,E.css)`
  font-size: 14px;
  line-height: 40px;
  font-weight: 500;
  display: block;
`;function cU(e,t,n){(0,b.useEffect)(()=>{let o;let r=window.scrollX,i=window.scrollY,a=ew(e=>{e instanceof TouchEvent&&!(e.changedTouches.length>1)&&(o=e.changedTouches[0].clientY)},"captureTouchStart"),s=ew(t=>{if(n?.(t)||!t.cancelable||!(t.target instanceof HTMLElement)||!e||t instanceof TouchEvent&&t.changedTouches.length>1)return;let r="down";t instanceof WheelEvent&&(r=t.deltaY>0?"down":"up"),t instanceof TouchEvent&&(r=o>t.changedTouches[0].clientY?"down":"up");let i=t.target,a=!1,s=!1;for(;i&&i!==document.body;){let t=i.scrollHeight-i.offsetHeight,n="up"===r&&i.scrollTop>0,o="down"===r&&i.scrollTop<t;if(t>0&&(o||n)&&(s=!0),i===e){a=!0;break}i=i?.parentElement}a&&s||t.preventDefault()},"preventScrollEvent"),l={passive:!1,capture:!1};return document.body.addEventListener("wheel",s,l),document.body.addEventListener("touchstart",a),document.body.addEventListener("touchmove",s,l),()=>{t||window.scrollTo(r,i),document.body.removeEventListener("wheel",s,l),document.body.removeEventListener("touchstart",a),document.body.removeEventListener("touchmove",s,l)}},[e,t,n])}ew(cU,"useDisableScrollOutside");var cW=ew(({className:e,onNavClose:t,highlight:n,globalNav:o,backgroundColor:r,screenState:i,dataset:a})=>{let s=(0,b.useRef)(null),l=(0,b.useRef)(null);return(0,b.useEffect)(()=>{"on"===i&&l.current?.focus()},[i]),a?.testid||(a={...a,testid:ce.navTopLevel}),cU(s.current),(0,C.BX)("section",{className:(0,E.cx)("sdsm-header",cQ,s7(r),e),...s6(a),children:[(0,C.BX)("div",{ref:s,className:cB,children:[(0,C.tZ)("header",{className:"sticky",children:(0,C.tZ)(lL,{size:"large",onClick:t,className:l2,iconName:"close",ref:l,"data-testid":ce.closeButton})}),(0,C.tZ)("nav",{className:"menu","data-testid":ce.navItems,children:o})]}),(0,C.tZ)("aside",{className:cZ,"data-testid":ce.navHighlights,children:n})]})},"GlobalNavScreenDesktop");cW.displayName="GlobalNavScreenDesktop";var cK=ew(({className:e,localNavMobile:t,localNavMobileFooter:n,globalNav:o,globalNavHeading:r,backgroundColor:i,showMobileGlobalLinks:a,dataset:s})=>{let l=(0,b.useRef)(null),{screenState:c}=(0,b.useContext)(lY);cU(l.current,"turning-off"===c);let u=ew(()=>(0,C.BX)("section",{className:lZ,children:[t,n&&(0,C.BX)(C.HY,{children:[(0,C.tZ)("hr",{}),n]})]}),"LocalNavSection");return s?.testid||(s={...s,testid:ce.navTopLevel}),(0,C.BX)("nav",{ref:l,className:(0,E.cx)("sdsm-header",cH,s7(i),e),...s6(s),children:[(t||n)&&(0,C.tZ)(u,{}),a&&(0,C.BX)("section",{className:(0,E.cx)(lq,s7(a9.Gray)),children:[(0,C.tZ)("label",{className:cG,children:r}),o]})]})},"GlobalNavScreenMobile");cK.displayName="GlobalNavScreenMobile";var cY=ew(e=>{let{screenState:t,mode:n}=(0,b.useContext)(lY);return(cP(),"off"===t)?null:n===ss.Mobile?(0,C.tZ)(cK,{...e,className:cR(t),screenState:t}):(0,C.tZ)(cW,{...e,className:cR(t),screenState:t})},"GlobalNavScreen");cY.displayName="GlobalNavScreen";var cX=(0,b.createContext)({lazy:!1}),cJ=ew((e,t)=>function(n){n&&e&&(n.src=e),n&&t&&(n.srcset=t)},"addSrcFactory"),c0=ew(({altText:e,className:t,style:n,defaultSrc:o,imgClassName:r,imgSrcs:i,height:a,width:s,fetchPriority:l,dataset:c})=>{let u=!i?.sources?.length,d=(0,b.useContext)(cX);return(0,C.BX)("picture",{className:t,style:n,children:[i?.sources?.map(e=>C.tZ("source",{srcSet:e.url,type:e.type,sizes:e.sizes},e.type)),(0,C.tZ)("img",{ref:cJ(i?.default??o,i?.defaultSrcSet),alt:e??"",className:r,sizes:i?.defaultSizes,height:a,width:s,src:u?i?.default??o:void 0,srcSet:u?i?.defaultSrcSet:void 0,loading:d?.lazy?"lazy":void 0,fetchpriorty:l,...s6(c)})]})},"Picture"),c1=(0,E.keyframes)`
  0% {
    transform: translateY(-50%) rotate(0deg);
  }

  100% {
    transform: translateY(-50%) rotate(360deg);
  }
`,c2="--spinner-size",c3=(0,E.css)`
  display: inline-block;
  position: relative;
  vertical-align: middle;
  z-index: ${99};
  width: var(${c2});
  height: var(${c2});

  &::before {
    width: calc(var(${c2}) / 2);
    height: var(${c2});
    border-radius: 0 var(${c2}) var(${c2}) 0;
  }

  &::before,
  &::after {
    border-width: calc(var(${c2}) / 8);
    border-style: solid;
    border-color: ${sc("--spinner-fg-color")};
    border-left: none;
    box-sizing: border-box;
    content: '';
    display: block;
    position: absolute;
    top: 50%;
    left: 50%;
    z-index: ${99};
    transform: translateY(-50%);
    transform-origin: 0% 50%;
    animation: ${c1} 1s linear 0s infinite;
  }

  &::after {
    width: calc(var(${c2}) / 4);
    height: calc(var(${c2}) / 2);
    border-radius: 0 calc(var(${c2}) / 2) calc(var(${c2}) / 2) 0;
    animation-direction: reverse;
  }
`,c5=ew(({size:e=16,className:t})=>(ls("sdsm-spinner"),(0,C.tZ)("div",{style:{[c2]:`${e}px`},className:(0,E.cx)("sdsm-spinner",c3,t)})),"Spinner"),c4=(0,E.css)`
  border-width: ${sc("--button-border-width")};
  border-style: solid;
  border-radius: ${sc("--button-border-radius")};
  cursor: pointer;
  /* Font-family on buttons defaults to Arial, so need to re-confirm. */
  font-family: ${sc("--font-family")};
  font-size: ${sc("--button-desktop-font-size")};
  font-weight: ${sc("--button-desktop-font-weight")};
  line-height: ${sc("--button-desktop-font-line-height")};
  text-decoration: none;
  transition: transform 150ms ease-in-out;
  white-space: nowrap;
  z-index: ${50};

  /* Preventing button from stretching containers.
     Important for mobile and extremely long text. */
  max-width: 100%;

  ${st} {
    font-size: ${sc("--button-mobile-font-size")};
    font-weight: ${sc("--button-mobile-font-weight")};
    line-height: ${sc("--button-mobile-font-line-height")};
  }

  /* Align icon and text */
  display: inline-flex;
  align-items: center;

  :disabled {
    opacity: 50%;
  }

  *[dir='rtl'] & {
    margin-left: ${sc("--spacing-xxs")};
  }

  box-shadow: none;

  &:hover:not([disabled]) {
    box-shadow: ${sc("--button-hover-shadow")};
    transform: translateY(-2px);
  }
  &:active:not([disabled]) {
    box-shadow: ${sc("--button-active-shadow")};
    transform: translateY(-1px);
  }
`,c6=(0,E.css)`
  &.button-compact {
    padding: ${sc("--button-compact-padding")};
  }
  &.button-regular {
    padding: ${sc("--button-regular-padding")};
  }
  &.button-large {
    padding: calc(${sc("--spacing-s")} - 1px) calc(${sc("--spacing-xxxl")} - 1px);
  }
  &.button-flat {
    padding: 0;
  }
`,c8=(0,E.css)`
  &.button-primary {
    background-color: ${sc("--button-primary-bg-color")};
    color: ${sc("--button-primary-fg-color")};
    border-color: ${sc("--button-primary-border-color")};
  }

  &.button-primary:hover {
    background-color: ${sc("--button-primary-hover-bg-color")};
    color: ${sc("--button-primary-fg-color")};
    border-color: ${sc("--button-primary-hover-border-color")};
  }

  &.button-secondary {
    background-color: ${sc("--button-secondary-bg-color")};
    color: ${sc("--button-secondary-fg-color")};
    border-color: ${sc("--button-secondary-border-color")};
  }

  &.button-secondary:hover {
    background-color: ${sc("--button-secondary-hover-bg-color")};
    color: ${sc("--button-secondary-fg-color")};
    border-color: ${sc("--button-secondary-hover-border-color")};
  }

  &.button-flat,
  &.button-flat:hover {
    background-color: transparent;
    color: ${sc("--button-flat-fg-color")};
    border-width: 0px;
    border-color: transparent;
    box-shadow: none;
    transform: none;
  }
`,c9=(0,E.css)`
  & > *:not(:last-child) {
    margin-right: ${sc("--spacing-xs")};
  }
  /* Spacing between items in the button */
  *[dir='rtl'] & > *:not(:last-child) {
    margin-right: unset;
    margin-left: ${sc("--spacing-xs")};
  }
`,c7=(0,E.css)`
  ${c4}
  ${c6}
  ${c8}
  ${c9}

  & > picture > img {
    height: 24px;
    display: block;
  }

  & > i {
    font-size: 22px;
    line-height: 18px;
  }
`,ue=(0,E.css)`
  overflow: hidden;
  text-overflow: ellipsis;
`,ut={Primary:"Primary",Secondary:"Secondary",Flat:"Flat"},un=(0,b.forwardRef)((e,t)=>{let{children:n,link:o,onClick:r,size:i=sl.Regular,type:a=ut.Secondary,image:s,iconName:l,className:c,loading:u,disabled:d,style:h,dataset:p,imgSrcs:f,imgAltText:m,buttonTextDataset:g,...v}=e;d&&o&&console.warn("You are trying to disable a button with a link. This does nothing.");let b=ew(e=>{r&&(r(e),e.preventDefault())},"onClick"),y=ew(e=>{"Enter"===e.key&&r&&(r(e),e.preventDefault())},"onKeyPress");ls("sdsm-button");let k={className:(0,E.cx)("sdsm-button",{"button-regular":i===sl.Regular,"button-compact":i===sl.Compact,"button-large":i===sl.Large,"button-primary":a===ut.Primary,"button-secondary":a===ut.Secondary,"button-flat":a===ut.Flat||i===sl.Flat},c7,c),onClick:b,onKeyPress:y,...o&&{href:o},style:h,disabled:u||d,...s6(p)},w={[ut.Flat]:"--button-flat-fg-color",[ut.Primary]:"--button-primary-fg-color",[ut.Secondary]:"--button-secondary-fg-color"},x=s||f?(0,C.tZ)(c0,{altText:m||s?.title,imgSrcs:f,defaultSrc:s?.url}):void 0,S=(0,C.BX)(C.HY,{children:[l&&(0,C.tZ)(lF,{name:l,fill:sc(w[a])}),x,n&&(0,C.tZ)("span",{className:ue,...s6(g),children:n})]});return o?(0,C.tZ)("a",{...v,...k,children:S}):(0,C.tZ)("button",{...v,...k,ref:t,children:u?(0,C.tZ)(c5,{}):S})});un.displayName="Button";var uo=(0,E.css)`
  ${sn} {
    font-size: ${sc("--action-desktop-font-size")};
    line-height: ${sc("--action-desktop-font-line-height")};
    font-weight: ${sc("--action-desktop-font-weight")};
  }

  ${st} {
    font-size: ${sc("--action-mobile-font-size")};
    line-height: ${sc("--action-mobile-font-line-height")};
    font-weight: ${sc("--action-mobile-font-weight")};
  }
`,ur=(0,E.css)`
  /* Reset <button> styles. */
  background: none;
  border: none;

  ${uo}

  background-color: ${sc("--dropdown-menu-bg-color")};
  border-radius: ${sc("--border-radius-m")};
  box-shadow: ${sc("--box-shadow-l")};
  color: ${sc("--foreground-color")};
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  min-width: 160px;
  padding: ${sc("--spacing-s")} ${sc("--spacing-m")};
`,ui=(0,E.css)`
  margin-left: ${sc("--spacing-s")};
  transform: rotate(0deg); /* 'down' */
  transition: transform 250ms;
  fill: ${sc("--foreground-color")};
  /** Aligning to where it was before. OK to move. */
  position: relative;
  top: 1px;
  left: 2px;
`,ua=(0,E.css)`
  transform: rotate(180deg); /* 'up' */
`,us=ew(({onClick:e,isExpanded:t,children:n,ariaLabel:o})=>(0,C.BX)("button",{className:(0,E.cx)("sdsm-dropdown-button",ur),onClick:e,role:"combobox","aria-expanded":t,"aria-haspopup":"listbox","aria-label":o,children:[n,(0,C.tZ)(lF,{size:20,className:(0,E.cx)(ui,{[ua]:t}),name:"chevron-down"})]}),"DropdownButton"),ul=(0,E.css)`
  border-radius: 0px 0px ${sc("--border-radius-s")} ${sc("--border-radius-s")};
  border-style: solid;
  border-width: 1px;
  border-color: ${sc("--dropdown-menu-border-color")};
  box-shadow: ${sc("--box-shadow-l")};
  margin-top: -4px;
  max-height: 50vh;
  min-width: 170px;
  overflow: auto;
  padding: calc(${sc("--dropdown-menu-padding")} / 2) 0;
  text-decoration: none;
`,uc=(0,E.css)`
  background-color: ${sc("--dropdown-menu-bg-color")};
  color: ${sc("--dropdown-item-fg-color")};
`,uu=(0,E.css)`
  ${ul}
  ${uc}
  position: absolute;
  width: max-content;
  max-width: 100%;
  z-index: ${2020};
`,ud=(0,E.css)`
  position: relative;
  display: inline-block;
`,uh=ew(e=>{if(-1!==e)return(0,E.css)`
    bottom: ${e}px;
    border-radius: ${sc("--border-radius-s")} ${sc("--border-radius-s")} 0 0;
  `},"getVerticalDirectionStyle"),up=(0,E.css)`
  left: 0;
`,uf=(0,E.css)`
  right: 0;
`,um=(0,E.css)`
  ${sn} {
    font-size: ${sc("--action-desktop-font-size")};
    line-height: ${sc("--action-desktop-font-line-height")};
    font-weight: ${sc("--action-desktop-font-weight")};
  }

  ${st} {
    font-size: ${sc("--action-mobile-font-size")};
    line-height: ${sc("--action-mobile-font-line-height")};
    font-weight: ${sc("--action-mobile-font-weight")};
  }
`,ug=(0,E.css)`
  color: ${sc("--dropdown-item-fg-color")};
  background-color: transparent;

  &.selected {
    color: ${sc("--dropdown-item-fg-active-color")};
    background-color: ${sc("--dropdown-item-bg-active-color")};
  }

  /* Has to be listed after selected so it can override. */
  &:hover {
    color: ${sc("--dropdown-item-fg-hover-color")};
    background-color: ${sc("--dropdown-item-bg-hover-color")};
  }
`,uv=(0,E.css)`
  /* Reset <button> style */
  display: block;
  border: none;

  padding: calc(${sc("--dropdown-menu-padding")} / 2) ${sc("--dropdown-menu-padding")};
  width: 100%;
  text-align: start;

  ${um}
  ${ug}

  position: relative;
  cursor: pointer;
`,ub=ew(({title:e,onClick:t,id:n,isSelected:o})=>(0,C.tZ)("button",{"data-test-id":n,"data-selected":!!o,className:(0,E.cx)("sdsm-dropdown-item",uv,{selected:o}),onClick:t,role:"option",children:e}),"DropdownMenuItem");ub.displayName="DropdownMenuItem";var uy=(0,b.forwardRef)(({items:e,onItemClick:t,className:n},o)=>(0,C.tZ)("div",{ref:o,className:(0,E.cx)(uu,"sdsm-dropdown",n),role:"listbox",children:e.map(({id:e,title:n,onClick:o,isSelected:r})=>(0,C.tZ)(ub,{id:e,isSelected:r,title:n??e,onClick:r=>{o?.(r),t({id:e,title:n})}},e))})),uk=ew(({items:e,selectedItemId:t,placeholderText:n="Select",onVisibleChange:o=P,onItemSelect:r,className:i,buttonComponent:a,ariaLabel:s,...l})=>{ls("sdsm-dropdown-menu");let[c,u]=(0,b.useState)(),[d,h]=(0,b.useState)(!1),p=(0,b.useRef)(d),f=(0,b.useRef)(null),m=(0,b.useRef)(null),g=(0,b.useRef)(-1),v=(0,b.useRef)(!1);(0,b.useEffect)(()=>{if(d&&d!==p.current&&m.current){let e=m.current.getBoundingClientRect();!v.current&&e.right>window.innerWidth&&(v.current=!0,m.current.style.right="0",m.current.style.left="unset"),v.current&&e.left<0&&(v.current=!1,m.current.style.right="unset",m.current.style.left="0")}},[d]),(0,b.useEffect)(()=>{p.current!==d&&(o(d),p.current=d)},[d,o]),(0,b.useEffect)(()=>{if(!d)return;let e=ew(e=>{e.target instanceof Element&&!(e.target&&f.current?.contains(e.target))&&h(!1)},"bodyClickListener");return document.body.addEventListener("click",e),()=>{document.body.removeEventListener("click",e)}},[d]);let y=(0,b.useCallback)(()=>{let e=f.current?.getBoundingClientRect(),t=document.documentElement.scrollHeight,n=window.scrollY;e?.bottom&&t-n-e.bottom<window.innerHeight/2?g.current=e.height:g.current=-1,h(e=>!e)},[h,f]),k=uh(g.current),w=(0,b.useCallback)(e=>{u(e),h(!1),r?.(e)},[r]),x=t?e.find(e=>e.id===t):c;return(0,C.BX)("div",{ref:f,className:(0,E.cx)("sdsm-dropdown-menu",ud,i),...l,children:[(0,C.tZ)(a??us,{isExpanded:d,onClick:y,ariaLabel:s,children:x?x?.title??eu(x?.id):n}),d&&(0,C.tZ)(uy,{ref:m,items:e.map(e=>({...e,isSelected:x?.id===e.id})),onItemClick:w,className:(0,E.cx)(v.current?uf:up,k)})]})},"DropdownMenu");(0,E.css)`
  background-color: ${sc("--footer-bar-bg-color")};
  border-top: 1px solid ${sc("--footer-bar-divider-border-color")};
  display: flex;
  justify-content: center;
`,(0,E.css)`
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  max-width: ${1440}px;
  padding-block: ${sc("--spacing-xl")};
  position: relative;
  white-space: nowrap;
  width: 100%;

  ${sn} {
    padding-block: ${sc("--spacing-s")};
    padding-inline: ${sc("--spacing-xl")};
  }

  ${sr} {
    gap: 27px;
    padding-block: ${sc("--spacing-l")};
    padding-inline: ${sc("--spacing-xxxxl")};
  }

  ${si} {
    flex-direction: row;
    gap: 0;
    padding-inline: ${sc("--spacing-xxxxl")};

    div {
      width: auto;
    }
  }

  ${st} {
    gap: 0;
    padding-inline: ${sc("--spacing-xl")};
  }
`,(0,b.createContext)({}),(0,E.css)`
  border-top-color: ${sc("--footer-border-color")};
  border-top-style: solid;
  border-top-width: 1px;
  background-color: ${sc("--footer-bg-color")};
  display: flex;
  justify-content: center;
`,(0,E.css)`
  box-sizing: border-box;
  display: flex;
  flex-wrap: wrap;
  gap: ${sc("--spacing-m")};
  max-width: ${1440}px;
  padding-block: ${sc("--spacing-xl")};
  padding-inline: ${sc("--spacing-l")};
  width: 100%;

  ${sr} {
    padding-inline: ${sc("--spacing-xxxxl")};
  }

  ${si} {
    padding-inline: ${sc("--spacing-xxxxl")};
  }
`,(0,E.css)`
  justify-content: space-between;
`,(0,E.css)`
  box-sizing: border-box;
`,(0,E.css)`
  align-items: center;
  border-top: 1px solid ${sc("--footer-divider-border-color")};
  background-color: ${sc("--footer-bg-color")};
  display: block;
  padding-block: ${sc("--spacing-xl")} ${sc("--spacing-xl")};
  padding-inline: ${sc("--spacing-xl")};
`,(0,E.css)`
  > div {
    width: 100%;
  }
`,(0,E.css)`
  > div {
    width: calc((100% - ${sc("--spacing-m")}) / 2);
  }
`,(0,E.css)`
  > div {
    width: calc((100% - (${sc("--spacing-m")} * 2)) / 3);
  }
`,(0,E.css)`
  > div {
    width: calc((100% - (${sc("--spacing-m")} * 3)) / 4);
  }
`,(0,E.css)`
  > div {
    width: calc((100% - (${sc("--spacing-m")} * 4)) / 5);
  }

  ${si} {
    > div {
      width: calc((100% - (${sc("--spacing-m")} * 5)) / 6);
    }
  }
`,(0,E.css)`
  font-size: 14px;
  color: ${sc("--foreground-color")};
  font-weight: 700;
  padding-inline-end: ${sc("--spacing-xl")};
  word-break: break-word;

  ${sn} {
    padding-inline-end: 0;
  }
`,(0,E.css)`
  margin-block-end: ${sc("--spacing-xs")};
`,(0,E.css)`
  border-bottom: 1px solid ${sc("--footer-border-color")};
  font-size: 14px;
  font-weight: 400;
  line-height: 16px;
  padding-block: ${sc("--spacing-m")};
  padding-inline: 0 ${sc("--spacing-m")};

  &:first-child {
    padding-block-start: ${sc("--spacing-s")};
  }

  ${sn} {
    border-bottom: 0;
    padding-bottom: ${sc("--spacing-xxs")};
    padding-inline: 0;
    padding-top: 0;

    &:first-child {
      padding-block-start: 0;
    }
  }
`,(0,E.css)`
  display: flex;
  flex-direction: column;
  list-style-type: none;
  margin: 0;
  padding-block: ${sc("--spacing-s")} 0;
  padding-inline: 0;
  width: 100%;

  ${sn} {
    padding: 0;
  }
`,(0,E.css)`
  a {
    margin-inline: 0;
    white-space: unset;
  }

  li {
    margin: 0;
  }

  align-items: flex-start;
  border: 0;
  display: flex;
  flex-direction: column;
  gap: ${sc("--spacing-m")};
  padding-block: ${sc("--spacing-s")};

  ${sn} {
    align-items: center;
    flex-direction: row;
    gap: ${sc("--spacing-l")};
    padding-block: 0;

    &:first-child {
      padding-block-start: 0;
    }
  }

  ${si} {
    &:first-child {
      margin-inline-start: 0;
    }

    &:last-child {
      margin-inline-end: 0;
    }
  }
`,(0,E.css)`
  margin: ${sc("--spacing-s")} 0;

  ${sn} {
    margin: ${sc("--spacing-xxs")} 0;
  }
`,(0,E.css)`
  color: ${sc("--foreground-color")};
  font-weight: 400;
  text-decoration: none;
  font-size: 14px;
  white-space: pre-wrap;

  @media (hover: hover) {
    &:hover {
      text-decoration: underline;
      text-decoration-thickness: ${sc("--spacing-xxxs")};
      text-underline-offset: ${sc("--spacing-xxxs")};
    }
  }

  ${sn} {
    margin: 0;
    margin-inline-end: ${sc("--spacing-s")};
  }
`;var uw=ew((e,t)=>n=>{[e,t].forEach(e=>{e&&("function"==typeof e?e(n):e.current=n)})},"combineRefs"),ux=(0,E.css)`
  width: 100%;
  height: 100%;
  display: block;
`,uS=(0,E.css)`
  object-fit: cover;
`,uC=(0,E.css)`
  object-fit: contain;
`,uF=ew(e=>e?uS:uC,"getObjectFit"),u_=(0,b.forwardRef)((e,t)=>{let{source:n,sourceType:o="video/mp4",posterSource:r,altText:i,className:a,style:s,muted:l=!0,autoPlay:c=!0,loop:u=!0,controls:d=!1,playsInline:h=!0,onPlay:p,captionsSource:f,isBackgroundVideo:m,onTimeUpdate:g,dataset:v}=e,y=(0,b.useRef)(null);(0,b.useEffect)(()=>{let e=y.current?.querySelector("track");if(y.current&&f&&!e){let e=document.createElement("track");e.setAttribute("src",f),e.setAttribute("kind","captions"),e.setAttribute("srclang","en"),e.setAttribute("label","English"),y.current.appendChild(e)}},[f]);let k=uF(m);return(0,C.tZ)("video",{className:(0,E.cx)("video",ux,k,a),style:s,poster:r,autoPlay:c,loop:u,muted:l,controls:d,playsInline:h,onPlay:p,preload:r?"none":"auto","aria-describedby":i,ref:uw(y,t),crossOrigin:"anonymous",onTimeUpdate:g,...s6(v),children:(0,C.tZ)("source",{src:n,type:o})},n)});u_.displayName="Video";var uN=ew(({children:e,className:t,maxHeight:n})=>{let o=(0,E.css)`
    ${n?`max-height: ${n}px;`:""}
    width: 100%;
    & > svg {
      /* Scaling the SVG */
      width: 100%;
    }
  `;return(0,C.tZ)("div",{className:(0,E.cx)(o,t),children:(0,C.BX)("svg",{viewBox:"0 0 654 366",width:"654",height:"366",fill:"none",xmlns:"http://www.w3.org/2000/svg","data-testid":"laptop-wrapper",children:[(0,C.tZ)("g",{filter:"url(#J)",fillRule:"evenodd",children:(0,C.tZ)("path",{d:"M95.626 1.127L104.459 1h444.227l8.834.127c9.637.157 17.141 7.735 \n          17.096 17.253.045-.115.285 5.943.285 8.833v292.992c0 1.781-.092 4.779-.172 \n          6.782-.05 1.237-.152 3.513-.238 4.224-1.023 8.479-8.201 14.964-16.971 \n          14.923.086.045-5.974.285-8.834.285H104.459c-2.861 0-8.92-.24-8.833-.285\n          -8.359.039-15.272-5.851-16.788-13.745-.208-1.083-.314-2.203-.309-3.351\n          -.045.114-.285-5.944-.285-8.833V27.214c0-2.876.24-8.948.285-8.833-.045-9.518 \n          7.591-17.299 17.097-17.253z",fill:"#303030"})}),(0,C.tZ)("g",{filter:"url(#L)",fillRule:"evenodd",children:(0,C.tZ)("path",{d:"M572.337 27.499l-.285-8.833c.045-7.935-6.31-14.449-14.247-14.404\n          -1.411-.051-6.09-.127-8.834-.127H104.459c-2.752 0-6.258.083-8.833.127\n          -7.937-.045-14.292 6.469-14.247 14.404-.057.246-.285 6.117-.285 8.833V322.12c0 \n          .889.037 2.314.088 3.582.059 1.457.223 3.895.304 4.478.768 5.534 5.499 9.421 \n          11.29 9.321-.039.112 4.331.285 6.269.285h454.77c2.043 0 6.421-.173 6.554-.285 \n          6.229.11 11.335-4.713 11.398-11.112-.06.036.112-4.348 0-6.269l.57-294.621z",fill:"#272727"})}),(0,C.BX)("g",{fillRule:"evenodd",children:[(0,C.tZ)("path",{d:"M574.988 216.885v103.069l-.144 6.217-.393 5.834c-1.491 7.885-8.427 \n          13.848-16.758 13.848l-8.884.24H104.133c-2.82 0-8.879-.24-8.879-.24-8.499 0\n          -15.286-6.207-16.53-14.328-.135-.879-.445-9.574-.445-11.552v-103.05l496.709\n          -.038z",fill:"url(#O)",fillOpacity:".03"}),(0,C.tZ)("path",{d:"M93.346 24.365h466.453c.355 0 .57.215.57.569v291.203a.52.52 0 0 1\n          -.57.569H93.346a.52.52 0 0 1-.57-.569V24.934c0-.355.215-.569.57-.569z",fill:"#111112"}),(0,C.tZ)("path",{d:"M272.029 350.802H100.508l-37.081-.284c-33.91.284-43.031-3.773-43.026\n          -4.748v-1.425h252.054l107.667.096h252.054v1.424c.004.976-9.117 5.033-43.027 \n          4.749-.622.019-37.081.284-37.081.284H380.546l-108.517-.096z",fill:"#1c1b1b"})]}),(0,C.tZ)("path",{d:"M21.255 336.082H631.32c.532 0 .855.313.855.57v9.118c0 .256-.323.57\n        -.855.57H21.255c-.532 0-.855-.314-.855-.57v-9.118c0-.257.322-.57.855-.57z",fill:"#1d1c1c",fillRule:"evenodd"}),(0,C.tZ)("foreignObject",{x:"92",y:"24",width:"468",height:"293",requiredFeatures:"http://www.w3.org/TR/SVG11/feature#Extensibility",children:e}),(0,C.BX)("defs",{children:[(0,C.BX)("filter",{id:"J",x:"77.244",y:"0",width:"498.657",height:"347.42",filterUnits:"userSpaceOnUse",colorInterpolationFilters:"sRGB",children:[(0,C.tZ)("feGaussianBlur",{stdDeviation:".5"}),(0,C.tZ)("feBlend",{in:"SourceGraphic",result:"C"}),(0,C.tZ)("feColorMatrix",{in:"SourceAlpha",values:"0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0",result:"D"}),(0,C.tZ)("feGaussianBlur",{stdDeviation:"1.5"}),(0,C.tZ)("feComposite",{in2:"D",operator:"arithmetic",k2:"-1",k3:"1"}),(0,C.tZ)("feBlend",{in2:"C"})]}),(0,C.BX)("filter",{id:"L",x:"79.093",y:"2.134",width:"495.243",height:"339.652",filterUnits:"userSpaceOnUse",colorInterpolationFilters:"sRGB",children:[(0,C.tZ)("feBlend",{in:"SourceGraphic",result:"C"}),(0,C.tZ)("feColorMatrix",{in:"SourceAlpha",values:"0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0",result:"D"}),(0,C.tZ)("feGaussianBlur",{stdDeviation:"2"}),(0,C.tZ)("feComposite",{in2:"D",operator:"arithmetic",k2:"-1",k3:"1"}),(0,C.tZ)("feBlend",{in2:"C",result:"E"}),(0,C.tZ)("feColorMatrix",{values:"0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.1 0"}),(0,C.tZ)("feBlend",{in2:"E"})]}),(0,C.BX)("linearGradient",{id:"O",x1:"111.017",y1:"227.897",x2:"111.017",y2:"340.072",gradientUnits:"userSpaceOnUse",children:[(0,C.tZ)("stop",{stopOpacity:".01"}),(0,C.tZ)("stop",{offset:"1"})]})]})]})})},"LaptopWrapper");uN.displayName="LaptopWrapper";var u$=(0,E.css)`
  display: block;
  object-fit: cover;
`,uE=(0,E.css)`
  ${u$}

  border-radius: ${8}px;
  height: auto;
  margin: auto;
  max-width: 100%;
  overflow: hidden;
  width: auto;
`,uO=(0,E.css)`
  ${u$}
  height: 100%;
  width: 100%;
`,uD=(0,E.css)`
  ${uO}
  object-fit: contain;
`,uT=(0,E.css)`
  ${uE}
  object-fit: contain;
`,uz=(0,E.css)`
  margin-bottom: ${16}px;
`,uM=(0,E.css)`
  ${st} {
    margin-top: 0;
  }
  ${so} {
    margin-top: 0;
  }
`,uI=(0,E.css)`
  display: flex;
  justify-content: center;
`,uP=(0,E.css)`
  background-image: url('https://images.ctfassets.net/kp51zybwznx4/5MBQ096OLy837zpHXtGsIa/0b35a1eadad459a767f935297483854c/iPhone.png');
  width: 250px;
  height: 530px;
  background-size: 100%;
  background-repeat: no-repeat;
  position: relative;
  margin-bottom: 20px;
  padding: 14px;
`,uj=ew(({children:e})=>(0,C.tZ)("div",{className:(0,E.cx)("phone-wrapper",uI),"data-testid":"phone-wrapper",children:(0,C.tZ)("div",{className:uP,children:e})}),"PhoneWrapper");uj.displayName="PhoneWrapper";var uR=(0,E.css)`
  position: fixed;
`,uL=(0,E.css)`
  width: 297px;
  height: 550px;
`,uA=(0,E.css)`
  border-radius: 15px;
`,uV=(0,E.css)`
  box-shadow: ${sc("--box-shadow-xl")};
  border-radius: ${sc("--spacing-m")};
`,uq=ew(({className:e,altText:t,imageSource:n,maxHeight:o,maxWidth:r,videoSource:i,sourceType:a,wrap:s=sa.None,marginBottom:l=!1,showVideoControls:c=!1,autoplay:u=!c,posterSource:d,onPlay:h,captionsSource:p,isBackgroundVideo:f,onTimeUpdate:m,imgSrcs:g,dataset:v,height:y,width:k})=>{let w=e;l&&(w=(0,E.cx)(w,uz));let x=null,{getLowEntropyHints:S}=(0,b.useContext)(a4),F=S(),_=!!("macOS"===F.platform&&F.browsers.find(e=>"Safari"===e.brand))&&s===sa.Phone,N=s===sa.Laptop||s===sa.Phone,$={maxHeight:o&&!N?`${o}px`:void 0,maxWidth:r&&!N?`${r}px`:void 0};if(n||g){let e=(0,E.cx)({[uO]:N,[uE]:!N,[uA]:s===sa.Phone,[uV]:s===sa.Shadow},w);g?x=(0,C.tZ)(c0,{imgSrcs:g,altText:t,imgClassName:e,style:$,dataset:v,height:y,width:k}):n&&(x=(0,C.tZ)(c0,{altText:t,imgClassName:e,style:$,defaultSrc:n,dataset:v,height:y,width:k}))}else{if(!i)return null;let e=f?uO:uD,n=f?uE:uT,o=(0,E.cx)({[uM]:s===sa.Phone,[e]:N,[uL]:s===sa.Phone&&_,[n]:!N,[uV]:s===sa.Shadow},w);x=(0,C.tZ)(u_,{source:i,sourceType:a,altText:t,className:o,style:$,autoPlay:u,playsInline:!c,loop:!c,muted:!c,controls:c,posterSource:d,onPlay:h,captionsSource:p,onTimeUpdate:m,isBackgroundVideo:f,dataset:v},i),_&&(x=(0,C.tZ)("div",{className:uR,children:x}))}return s===sa.Laptop?(0,C.tZ)(uN,{className:w,maxHeight:o,children:x}):s===sa.Phone?(0,C.tZ)(uj,{className:w,maxHeight:o,children:x}):(0,C.tZ)("div",{"data-testid":"no-device-wrapper",children:x})},"Media");function uZ(){return lA()===ss.Mobile}uq.displayName="Media",ew(uZ,"useIsMobile");var uB={[a8.Top]:(0,E.css)`
    left: 50%;
    top: 0;
    transform: translateX(-50%);
  `,[a8.Bottom]:(0,E.css)`
    left: 50%;
    bottom: 0;
    transform: translateX(-50%);
  `,[a8.Middle]:(0,E.css)`
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
  `},uQ=(0,E.css)`
  position: fixed;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
  box-shadow: none;
  z-index: ${1999};
  overflow: hidden;
  background: ${sc("--modal-background-color")};
  backdrop-filter: blur(4px);

  /* ==========================================================================================================
  Specific fix to address errant behavior in Safari browser when in fullscreen mode.
    Bug Behavior: certain CSS transforms (like blur) will cause the browser to stop honoring z-index values.
      Since this style is only applied when in full-screen mode, there should be no issues with
      removing the background blur since it should be hidden by the fullscreen content.
    Use Case: Rendering a Video player inside of the SDS-M Modal component.
  ========================================================================================================== */
  :-webkit-full-screen-ancestor:not(iframe) {
    /* NOTE: Safari needs the prefix. See https://caniuse.com/css-backdrop-filter */
    /* stylelint-disable-next-line property-no-vendor-prefix */
    -webkit-backdrop-filter: unset;
  }
`,uH=(0,E.css)`
  position: fixed;
`,uG=(0,E.css)`
  --close-button-size: 40px;

  ${st} {
    right: 0;
    --close-button-size: 32px;
  }

  /* Render over video content. */
  z-index: ${2e3};

  *[dir='rtl'] {
    left: -80px;
  }

  position: absolute;
  right: calc(0px - calc(var(--close-button-size) + ${sc("--spacing-s")}));
  top: calc(0px - calc(var(--close-button-size) + ${sc("--spacing-s")}));
`,uU=ew(({className:e,contentClassName:t,isBlocking:n=!1,verticalAlignment:o=a8.Middle,onClose:r,isDisplayed:i=!1,portalRoot:a,style:s,dataset:l,children:c,disableBackgroundScroll:u,shouldAllowScrollEvent:d})=>{ls("sdsm-modal");let[h,p]=(0,b.useState)(null),f=uZ(),m=(0,b.useCallback)(e=>p(e),[p]);cU(n||u?h:null,void 0,d);let g=(0,C.tZ)("section",{className:(0,E.cx)("sdsm-modal",uQ,{"sdsm-modal-blocking":n},e),style:s,...s6(l),children:(0,C.BX)("div",{className:(0,E.cx)(uH,"sdsm-modal-content",uB[o],t),ref:m,children:[c,!n&&(0,C.tZ)(lL,{size:f?"medium":"large",className:uG,iconName:"cross",onClick:r})]})});return a?i?(0,ed.createPortal)(g,a):null:i?g:null},"Modal"),uW=(0,E.css)`
  display: inline-flex;

  & input {
    height: 0;
    width: 0;
    visibility: hidden;
  }

  & label {
    display: flex;
    align-items: center;
    justify-content: space-between;
    cursor: pointer;
    width: 50px;
    height: 29px;
    background: ${sc("--toggle-slider-background-color")};
    border-radius: 100px;
    position: relative;
  }

  & span {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 25px;
    height: 25px;
    border-radius: 45px;
    transition: 0.2s;
    background: ${sc("--toggle-slider-switch-color")};
    box-shadow: 0px 0px 5px 1px rgb(10 10 10 / 29%);
  }

  ${st} {
    & span {
      width: 12px;
      height: 12px;
    }

    & label {
      width: 28px;
      height: 16px;
    }
  }
`,uK=(0,E.css)`
  display: inline-flex;

  & label span {
    transform: translateX(calc(100% - 4px));
  }

  & label {
    background-color: ${sc("--toggle-slider-active-color")};
  }

  ${st} {
    & input:checked + label span {
      transform: translateX(calc(100% - 0.5px));
    }
  }
`,uY=ew(({isChecked:e,className:t,onToggle:n,id:o})=>(ls("sdsm-toggle-slider"),(0,C.tZ)("div",{className:(0,E.cx)("sdsm-toggle-slider",uW,{[uK]:e},t),children:(0,C.BX)("label",{children:[(0,C.tZ)("input",{onChange:n,checked:e,id:o,type:"checkbox"}),(0,C.tZ)("span",{})]})})),"ToggleSlider");uY.displayName="ToggleSlider";var uX=(0,E.css)`
  fill: #fff;
`,uJ=ew(({className:e,width:t,height:n,backgroundColor:o})=>(0,C.tZ)("svg",{version:"1.1",xmlns:"http://www.w3.org/2000/svg",x:"0px",y:"0px",viewBox:"0 0 500 500",className:e,width:t,height:n,children:(0,C.tZ)("g",{id:"Layer_1",children:(0,C.BX)("g",{children:[(0,C.tZ)("g",{children:(0,C.tZ)("path",{className:o!==a9.Black?uX:void 0,d:"M484.6,369.3c-2-6.8-11.9-11.6-11.9-11.6l0,0c-0.9-0.5-1.7-0.9-2.4-1.3c-16.4-7.9-30.8-17.4-43.1-28.2\n				c-9.8-8.7-18.2-18.2-25-28.4c-8.2-12.4-12.1-22.8-13.8-28.4c-0.9-3.7-0.8-5.1,0-7c0.6-1.6,2.5-3.1,3.4-3.8\n				c5.5-3.9,14.4-9.7,19.9-13.2c4.7-3.1,8.8-5.7,11.2-7.4c7.7-5.4,12.9-10.8,16-16.7c4-7.6,4.5-16,1.4-24.3\n				c-4.2-11.1-14.6-17.8-27.8-17.8c-2.9,0-6,0.3-9,1c-7.6,1.6-14.8,4.3-20.8,6.7c-0.4,0.2-0.9-0.2-0.9-0.6\n				c0.6-14.9,1.3-34.9-0.3-53.9c-1.5-17.2-5-31.7-10.8-44.3c-5.8-12.7-13.4-22.1-19.3-28.9c-5.7-6.5-15.5-16-30.5-24.5\n				c-21-12-45-18.1-71.1-18.1c-26.1,0-50,6.1-71.1,18.1c-15.8,9-25.9,19.2-30.5,24.5c-5.9,6.8-13.5,16.2-19.3,28.9\n				c-5.8,12.6-9.3,27.1-10.8,44.3c-1.6,19.1-1,37.5-0.3,53.9c0,0.5-0.5,0.8-0.9,0.6c-6-2.3-13.2-5-20.7-6.7c-3-0.7-6-1-9-1\n				c-13.2,0-23.6,6.7-27.8,17.8c-3.1,8.3-2.7,16.7,1.4,24.3c3.1,5.9,8.4,11.4,16,16.7c2.4,1.7,6.4,4.3,11.2,7.4\n				c5.3,3.5,14,9.1,19.5,13c0.7,0.5,3,2.3,3.8,4.1c0.8,2,0.9,3.4-0.1,7.3c-1.7,5.7-5.6,15.9-13.7,28.1c-6.8,10.2-15.2,19.7-25,28.4\n				C60.5,339,46,348.5,29.7,356.5c-0.8,0.4-1.7,0.8-2.7,1.4l0,0c0,0-9.8,5-11.6,11.4c-2.7,9.5,4.5,18.4,11.9,23.2\n				c12.1,7.8,26.9,12,35.4,14.3c2.4,0.6,4.5,1.2,6.5,1.8c1.2,0.4,4.3,1.6,5.6,3.3c1.7,2.1,1.9,4.8,2.5,7.8v0c0.9,5,3,11.2,9.2,15.5\n				c6.8,4.7,15.5,5,26.4,5.5c11.5,0.4,25.8,1,42.1,6.4c7.6,2.5,14.4,6.7,22.4,11.6c16.6,10.2,37.2,22.9,72.5,22.9\n				c35.3,0,56.1-12.8,72.8-23c7.9-4.8,14.7-9,22.1-11.5c16.3-5.4,30.6-5.9,42.1-6.4c11-0.4,19.6-0.7,26.4-5.5\n				c6.6-4.6,8.6-11.4,9.4-16.6c0.5-2.5,0.8-4.8,2.3-6.7c1.2-1.6,4.1-2.7,5.4-3.2c2-0.6,4.2-1.2,6.7-1.9c8.6-2.3,19.3-5,32.3-12.4\n				C485.2,385.6,486.3,374.7,484.6,369.3z"})}),(0,C.tZ)("path",{className:o===a9.Black?uX:void 0,d:"M498.2,364c-3.5-9.5-10.1-14.5-17.6-18.7c-1.4-0.8-2.7-1.5-3.8-2c-2.2-1.2-4.5-2.3-6.8-3.5\n			c-23.5-12.5-41.8-28.2-54.6-46.8c-4.3-6.3-7.3-12-9.4-16.6c-1.1-3.1-1-4.9-0.3-6.5c0.6-1.2,2.2-2.5,3-3.1c4-2.7,8.2-5.4,11-7.2\n			c5-3.3,9-5.8,11.6-7.6c9.6-6.7,16.4-13.9,20.6-21.9c5.9-11.3,6.7-24.2,2.1-36.3c-6.4-16.8-22.3-27.3-41.5-27.3\n			c-4,0-8,0.4-12.1,1.3c-1.1,0.2-2.1,0.5-3.1,0.7c0.2-11.4-0.1-23.6-1.1-35.6c-3.6-42-18.3-64-33.7-81.6c-6.4-7.3-17.5-18-34.2-27.6\n			C305.1,10.6,278.8,3.9,250,3.9c-28.7,0-55,6.7-78.3,20c-16.8,9.6-27.9,20.3-34.3,27.6c-15.3,17.5-30,39.6-33.7,81.6\n			c-1,11.9-1.3,24.1-1.1,35.6c-1-0.3-2.1-0.5-3.1-0.7c-4-0.9-8.1-1.3-12.1-1.3c-19.3,0-35.2,10.4-41.5,27.3\n			c-4.6,12.1-3.8,25,2.1,36.3c4.2,8,11,15.2,20.6,21.9c2.6,1.8,6.6,4.4,11.6,7.6c2.7,1.8,6.7,4.3,10.6,6.9c0.6,0.4,2.7,1.9,3.4,3.4\n			c0.8,1.7,0.8,3.5-0.4,6.8c-2.1,4.6-5,10.1-9.2,16.3c-12.4,18.2-30.3,33.6-53,46c-12,6.4-24.6,10.6-29.9,25\n			c-4,10.9-1.4,23.2,8.8,33.6l0,0c3.3,3.6,7.5,6.8,12.8,9.7c12.4,6.9,23,10.2,31.3,12.5c1.5,0.4,4.8,1.5,6.3,2.8\n			c3.7,3.2,3.2,8.1,8.1,15.2c3,4.4,6.4,7.4,9.2,9.4c10.3,7.1,21.9,7.6,34.2,8c11.1,0.4,23.7,0.9,38.1,5.7c6,2,12.1,5.8,19.3,10.2\n			c17.2,10.6,40.8,25,80.2,25c39.4,0,63.1-14.5,80.5-25.2c7.1-4.4,13.3-8.1,19-10c14.4-4.7,27-5.2,38.1-5.7\n			c12.3-0.5,23.9-0.9,34.2-8c3.2-2.2,7.3-5.9,10.5-11.5c3.5-6,3.4-10.2,6.8-13.2c1.4-1.2,4.3-2.2,5.9-2.7\n			c8.4-2.3,19.1-5.7,31.7-12.6c5.6-3.1,10-6.5,13.4-10.3c0,0,0.1-0.1,0.1-0.1C499.7,386.6,502.1,374.6,498.2,364z M463.2,382.8\n			c-21.4,11.8-35.6,10.5-46.6,17.7c-9.4,6-3.8,19.1-10.7,23.8c-8.4,5.8-33.2-0.4-65.1,10.2c-26.4,8.7-43.2,33.8-90.7,33.8\n			c-47.6,0-64-25-90.7-33.8c-32-10.6-56.8-4.4-65.1-10.2c-6.8-4.7-1.3-17.7-10.7-23.8c-11-7.1-25.3-5.9-46.6-17.7\n			c-13.6-7.5-5.9-12.2-1.4-14.4c77.4-37.5,89.8-95.4,90.3-99.7c0.7-5.2,1.4-9.3-4.3-14.6c-5.5-5.1-30.1-20.3-36.9-25\n			c-11.3-7.9-16.2-15.7-12.6-25.4c2.5-6.7,8.8-9.2,15.4-9.2c2,0,4.1,0.2,6.1,0.7c12.4,2.7,24.4,8.9,31.3,10.6c1,0.2,1.8,0.3,2.6,0.3\n			c3.7,0,5-1.9,4.8-6.1c-0.8-13.5-2.7-39.9-0.6-64.5c2.9-33.9,13.9-50.7,26.9-65.6c6.2-7.1,35.5-38.1,91.6-38.1\n			c56.2,0,85.3,30.9,91.6,38.1c13,14.9,23.9,31.7,26.9,65.6c2.1,24.6,0.3,51-0.6,64.5c-0.3,4.5,1.1,6.1,4.8,6.1\n			c0.8,0,1.6-0.1,2.6-0.3c6.9-1.7,19-7.9,31.3-10.6c2-0.4,4.1-0.7,6.1-0.7c6.6,0,12.8,2.5,15.4,9.2c3.7,9.7-1.3,17.5-12.6,25.4\n			c-6.8,4.7-31.3,19.9-36.9,25c-5.7,5.3-5,9.4-4.3,14.6c0.6,4.4,12.9,62.2,90.3,99.7C469.1,370.6,476.8,375.3,463.2,382.8z"})]})})}),"LogoLight"),u0=(0,E.css)`
  text-decoration: none;
  display: flex;
  padding-top: 0;
  padding-bottom: 0;
  cursor: pointer;
`,u1=(0,E.css)`
  max-width: 100%;
  /* Prevents large logo images from exceeding header height */
  max-height: calc(${64}px - ${sc("--spacing-m")});

  ${st} {
    max-width: 100%;
  }
  display: block;
`,u2=(0,E.css)`
  margin-left: ${sc("--spacing-m")};

  *[dir='rtl'] & {
    margin-right: ${sc("--spacing-m")};
    margin-left: 0;
  }

  ${lv}
`,u3=ew(({imgSrcs:e,imageSource:t,label:n,url:o="/",onClick:r,backgroundColor:i,className:a})=>{let s;let l=ew(e=>{r&&(r(e),e.preventDefault())},"onClickWrapped");return s=e?(0,C.tZ)(c0,{altText:"logo",imgClassName:u1,imgSrcs:e}):t?(0,C.tZ)(c0,{altText:"logo",imgClassName:u1,defaultSrc:t}):(0,C.tZ)(uJ,{className:a,width:32,height:32,backgroundColor:i}),(0,C.BX)("a",{href:o,onClick:l,className:u0,children:[s,n&&(0,C.tZ)("span",{className:u2,children:n})]})},"Ghost");u3.displayName="Ghost";var u5="navigator-item-click",u4=(0,E.css)`
  font-weight: 700;
  color: ${sc("--global-header-navigator-item-active-color")};

  /* Taken from reactMenuCss - note the menu itself is not themed. The background is always white.*/
  .szh-menu__item--active,
  &.szh-menu__item--active {
    color: #fff;
    background-color: ${"#E9EAEB"};
  }
`,u6=(0,E.css)`
  margin: ${sc("--spacing-xxs")};
  /* TODO: Font-weight should use motif vars, but first we need to change the active page indicator. https://jira.sc-corp.net/browse/ENTWEB-8894 */
  font-weight: 400;
  font-size: 13px;
  color: ${sc("--global-header-navigator-item-color")};

  :hover,
  :focus {
    background-color: ${"#E9EAEB"};
    border-radius: ${sc("--spacing-xxs")};
  }
`,u8=(0,E.css)`
  color: ${sc("--global-header-navigator-item-color")};
  cursor: pointer;
  display: flex;
  font-size: ${sc("--action-desktop-font-size")};
  font-weight: 500;
  text-decoration: none;
  white-space: nowrap;

  ${st} {
    font-size: ${sc("--action-mobile-font-size")};
  }
`,u9=(0,E.css)`
  ${u8}
  align-items: center;
  line-height: ${64}px;
  margin-left: ${sc("--spacing-m")};
  margin-right: ${sc("--spacing-m")};
  padding-bottom: 0;
  padding-top: 0;

  &.${u5} {
    cursor: pointer;

    &:hover {
      color: ${sc("--global-header-navigator-item-hover-color")};
    }

    & span {
      position: relative; /* For the underline */
      overflow-y: hidden;
    }

    /** Custom underline */
    & span::after {
      /* TODO: consolidate the rounded underline css? */
      background-color: ${sc("--global-header-navigator-item-hover-color")};
      bottom: -8px;
      content: '';
      height: 8px;
      left: 0;
      position: absolute;
      border-radius: ${sc("--border-radius-s")};
      transform: translateY(0);
      transition: all 0.15s ease;
      width: 100%;
    }

    &:hover span::after {
      opacity: 0.6; /* TODO: convert to motif variable instead */
      transform: translateY(-4px);
    }

    & span.selected {
      color: ${sc("--global-header-navigator-item-active-color")};
    }

    & span.selected::after {
      opacity: 1;
      transform: translateY(-4px);
    }
  }
`,u7=s9`
  &.szh-menu-container,
  .szh-menu-container {
    position: relative;
    width: 0px;
    height: 0px;
  }

  .szh-menu {
    margin: 0;
    padding: 0;
    list-style: none;
    box-sizing: border-box;
    width: max-content;
    position: absolute;
    z-index: ${2020};
    border: 1px solid rgba(0, 0, 0, 0.1);
    background-color: #fff;
  }
  .szh-menu:focus {
    outline: none;
  }
  .szh-menu--state-closed {
    display: none;
  }
  .szh-menu__arrow {
    box-sizing: border-box;
    width: 0.75rem;
    height: 0.75rem;
    background-color: #fff;
    border: 1px solid transparent;
    border-left-color: rgba(0, 0, 0, 0.1);
    border-top-color: rgba(0, 0, 0, 0.1);
    position: absolute;
    z-index: -1;
  }
  .szh-menu__arrow--dir-left {
    right: -0.375rem;
    transform: translateY(-50%) rotate(135deg);
  }
  .szh-menu__arrow--dir-right {
    left: -0.375rem;
    transform: translateY(-50%) rotate(-45deg);
  }
  .szh-menu__arrow--dir-top {
    bottom: -0.375rem;
    transform: translateX(-50%) rotate(-135deg);
  }
  .szh-menu__arrow--dir-bottom {
    top: -0.375rem;
    transform: translateX(-50%) rotate(45deg);
  }
  .szh-menu__item {
    cursor: pointer;
  }
  .szh-menu__item:focus {
    outline: none;
  }
  .szh-menu__item--hover {
    background-color: #ebebeb;
  }
  .szh-menu__item--focusable {
    cursor: default;
    background-color: inherit;
  }
  .szh-menu__item--disabled {
    cursor: default;
    color: #aaa;
  }
  .szh-menu__submenu {
    position: relative;
  }
  .szh-menu__group {
    box-sizing: border-box;
  }
  .szh-menu__radio-group {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .szh-menu__divider {
    height: 1px;
    margin: 0.5rem 0;
    background-color: rgba(0, 0, 0, 0.12);
  }

  .szh-menu-button {
    box-sizing: border-box;
  }

  .szh-menu {
    user-select: none;
    color: #212529;
    border: none;
    border-radius: 0.25rem;
    box-shadow: 0 3px 7px rgba(0, 0, 0, 0.133), 0 0.6px 2px rgba(0, 0, 0, 0.1);
    min-width: 10rem;
    padding: 0.5rem 0;
  }
  .szh-menu__item {
    display: flex;
    align-items: center;
    position: relative;
    padding: 0.375rem 1.5rem;
  }
  .szh-menu-container--itemTransition .szh-menu__item {
    transition-property: background-color, color;
    transition-duration: 0.15s;
    transition-timing-function: ease-in-out;
  }

  .szh-menu__item--type-radio {
    padding-left: 2.2rem;
  }
  .szh-menu__item--type-radio::before {
    content: '\\25CB';
    position: absolute;
    left: 0.8rem;
    top: 0.55rem;
    font-size: 0.8rem;
  }
  .szh-menu__item--type-radio.szh-menu__item--checked::before {
    content: '\\25CF';
  }
  .szh-menu__item--type-checkbox {
    padding-left: 2.2rem;
  }
  .szh-menu__item--type-checkbox::before {
    position: absolute;
    left: 0.8rem;
  }
  .szh-menu__item--type-checkbox.szh-menu__item--checked::before {
    content: '\\2714';
  }
  .szh-menu__submenu > .szh-menu__item {
    padding-right: 2.5rem;
  }
  .szh-menu__submenu > .szh-menu__item::after {
    content: '\\276F';
    position: absolute;
    right: 1rem;
  }
  .szh-menu__header {
    color: #888;
    font-size: 0.8em;
    padding: 0.2rem 1.5rem;
    text-transform: uppercase;
  }

  *[dir='rtl'] .szh-menu--dir-right {
    left: -10.3rem !important;
  }

  *[dir='rtl'] .szh-menu--dir-left {
    left: -12rem !important;
  }
`,de=(0,E.css)`
  text-decoration: underline;
`,dt=(0,E.css)`
  /**
   * use negative margin to make icon take up space same as previous
   */
  margin: calc(${sc("--spacing-xxs")} * -1);
  fill: ${sc("--global-header-navigator-item-color")};

  margin-left: calc(${sc("--spacing-xs")} - 4px);
  *[dir='rtl'] & {
    margin-right: calc(${sc("--spacing-xs")} - 4px);
  }
`;function dn(e){return!!e.id}ew(dn,"isItemProps");var dr=ew((e,t)=>b.Children.map(e,n=>{if(e&&(0,b.isValidElement)(n)&&n&&n.props&&dn(n.props)){let e={...n.props};return t&&(e.id=`${t}.${e.id}`),n.props.isSelected&&(e.title=(0,C.tZ)("span",{className:de,children:e.title})),e}}),"getItemProps"),di=ew((e,t)=>{let{id:n,title:o,isSelected:r,children:i,onClick:a}=e,s=0===b.Children.count(i),l=t?.has(n)??!1,c=r??l;return s?(0,C.tZ)(eh.s,{onClick:e=>a&&a(e.syntheticEvent),className:(0,E.cx)(u6,{[u4]:c}),children:o},n):(0,C.tZ)(ep.W,{label:o,itemProps:{onClick:a},className:(0,E.cx)(u6,{[u4]:c}),children:b.Children.map(i??[],({props:e})=>di(e,t))},n)},"renderChildrenWithBreadcrumbs");(0,E.injectGlobal)(u7);var da=ew(({id:e,children:t,url:n,onClick:o,isSelected:r,title:i,dataset:a})=>{ls("sdsm-header");let{navigationBreadcrumbs:s}=(0,b.useContext)(lY),[l,c]=(0,b.useState)(r??!1);(0,b.useEffect)(()=>{let t=s?.has(e)??!1;c(void 0===r?t:r)},[e,r,s]);let[u,d]=(0,b.useState)(!1),h=(0,b.useRef)(null),[p,f]=(0,b.useState)("first"),m=o||n?u5:null,g=(0,C.tZ)("span",{className:(0,E.cx)({selected:l}),children:i}),v=dr(t,e),y=ew(e=>{e.stopPropagation(),"Enter"===e.key||" "===e.key||"ArrowDown"===e.key?(d(!0),f("first")):"ArrowUp"===e.key&&(d(!0),f("last"))},"handleKeyPress"),k=ew(e=>{o&&(o(e),e.preventDefault())},"onClickWrapped");return v&&em(v)?(0,C.BX)("section",{className:"sdsm-header",children:[(0,C.BX)("a",{href:n,className:(0,E.cx)(u9,m),onClick:e=>{d(!0),k(e)},ref:h,onMouseEnter:()=>d(!0),onMouseLeave:()=>d(!1),onKeyDown:y,tabIndex:0,...s6(a),children:[g,(0,C.tZ)(lF,{size:24,className:dt,name:u?"chevron-up":"chevron-down"})]}),(0,C.tZ)(ef.B,{className:s7(a9.White),state:u?"open":"closed",anchorRef:h,onMouseEnter:()=>d(!0),onMouseLeave:()=>d(!1),onClose:()=>d(!1),menuItemFocus:{position:p},children:b.Children.map(t??[],({props:e})=>di(e,s))})]}):(0,C.tZ)("a",{href:n,className:(0,E.cx)("sdsm-header",u9,m),onClick:k,tabIndex:0,...s6(a),children:g})},"ItemDesktop");da.displayName="Navigator.Item (Desktop)";var ds=ew(({imageUrl:e,settings:t,currentUrl:n})=>{if(!t)return e;let{progressive:o,width:r,height:i,format:a,quality:s=40,fit:l}=t,c=new URL(e,n);return a&&c.searchParams.set("fm",a),l&&c.searchParams.set("fit",l),s>0&&s<=100?c.searchParams.set("q",String(s)):c.searchParams.set("q",String(40)),o&&c.searchParams.set("fl","progressive"),r&&0<r&&r<4e3&&c.searchParams.set("w",String(r)),i&&0<i&&i<4e3&&c.searchParams.set("h",String(i)),c.href},"getImageUrl"),dl=ew((e,t,n)=>t?.size?t.size.sizeToUrl.map(({size:t,settings:o})=>t?`${ds({imageUrl:e,settings:{...o,format:n??o?.format}})} ${t}`:ds({imageUrl:e,settings:{...o,format:n??o?.format}})).join(","):ds({imageUrl:e,settings:{...t?.image,format:n??t?.image?.format,quality:t?.quality}}),"getSrcSetUrl"),dc=ew(e=>{let{isSlowConnection:t,supportsAvif:n,supportsWebP:o}=e;return{getImageSources:(0,b.useCallback)((e,o)=>{if(!e)return;if(o&&t&&(o.quality=10),e.toLowerCase().endsWith(".svg"))return{sources:[],default:e};let r=[{type:"image/avif",url:dl(e,o,"avif"),sizes:o?.size?.sizes},{type:"image/webp",url:dl(e,o,"webp"),sizes:o?.size?.sizes}];n||(r=r.filter(e=>"image/avif"!==e.type));let i={image:o?.image};return o?.size&&(i={image:o.size.sizeToUrl.reduce((e,t)=>e&&parseInt(e.size)<parseInt(t.size)?e:t).settings}),{sources:r,default:dl(e,i)}},[t,n]),getBestBgImgSrc:(0,b.useCallback)((e,t)=>{if(!e)return;if(e.toLowerCase().endsWith(".svg"))return e;let r=n&&"avif"||o&&"webp"||"jpg",i={...t};return r&&(i.format=r),ds({imageUrl:e,settings:i})},[n,o])}},"useContentfulImages");(0,b.createContext)({controller:e_});var du=(0,E.css)`
  .sdsm-dropdown-button {
    font-size: 14px;
    line-height: 20px;
  }
`,dd=ew(({currentLocale:e,supportedLocales:t,onLocaleChange:n,className:o})=>{let r=(0,b.useContext)(aq);e??(e=r.currentLocale),t??(t=r.supportedLocales??aA),n??(n=r.onLocaleChange);let i=Object.values(t),a=(0,b.useCallback)(e=>{let t=e.id;N.set("sc-language",t,{domain:eE(window.location.host),secure:!0}),r.onEvent?.({action:eS.LocaleSelect,component:"LocaleDropdown",label:`Language: ${t}`}),n?.(t)},[r,n]),s=i.map(e=>{let t=eN[e.code]??e.name;return{id:e.code,title:t}});return(0,C.tZ)(uk,{items:s,className:(0,E.cx)(du,o),"data-testid":"gc-locale-language",selectedItemId:e??aL,onItemSelect:a})},"LocaleDropdown");dd.displayName="LocaleDropdown";var dh=(0,E.css)`
  font-family: 'Graphik', Helvetica, sans-serif;

  * {
    box-sizing: border-box;
    margin: 0;
  }
`,dp=(0,E.css)`
  ${dh}

  /* override site settings that override header behaviors */
  h1,h2,h3,h4,h5,h6 {
    font-family: 'Graphik', Helvetica, sans-serif;
    text-transform: initial;
    line-height: inherit;
  }

  /* Ensure consistent rendering of lists */
  ul {
    list-style: disc outside;
  }
`,df=(0,E.css)`
  .modal-header {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    margin-bottom: ${sc("--spacing-xs")};

    .logo-container {
      display: flex;
      justify-content: center;
      flex-direction: column;
    }
  }
`,dm=(0,E.css)`
  .contentful-rich-text {
    font-weight: 400;
    line-height: 26px;

    h3 {
      font-size: 1.17em;
    }

    a {
      color: inherit;
      font-weight: 500;
    }
    > ul {
      padding-left: ${sc("--spacing-l")};
      margin-bottom: ${sc("--spacing-s")};
    }
    > ul > li {
      margin-bottom: ${sc("--spacing-xxs")};
    }
    > p {
      margin: 0 0 ${sc("--spacing-xxs")};
    }
    *[dir='rtl'] & {
      > ul {
        padding-left: unset;
        padding-right: ${sc("--spacing-l")};
      }
    }
  }
`,dg=(0,E.css)`
  .${s7(a9.Black)} .category-border.category-border {
    /* Hardcoding Enum colors because we don't have a relevant CSS variable to use. */
    border: 1px solid ${"#53575B"};
  }
`,dv=(0,E.css)`
  .category-border {
    /* Hardcoding Enum colors because we don't have a relevant CSS variable to use. */
    border: 1px solid ${"#F7F8F9"};
    border-radius: ${sc("--border-radius-m")};
    padding: ${sc("--spacing-s")};
    margin-bottom: ${sc("--spacing-s")};
  }

  .category-title-container {
    display: flex;
    margin-bottom: ${sc("--spacing-xs")};
  }
  .category-title {
    flex: 1;

    ${lg}

    *[dir='rtl'] & {
      text-align: right;
    }
  }
  .category-status {
    font-size: 14px;
    margin-right: ${sc("--spacing-xs")};
    *[dir='rtl'] & {
      margin-right: unset;
      margin-left: ${sc("--spacing-xs")};
    }
  }
  .category-toggle {
    display: flex;
    justify-content: center;
    flex-direction: column;
  }
`,db=(0,E.css)`
  .modal-footer {
    padding-top: ${sc("--spacing-s")}; /* Padding set by design */
    text-align: center;
    display: flex;
    /* Reverse direction so that when we wrap on smaller screens, the break happens on 1st, rather than last element.
       NOTE: requires that content be rendered in reverse order */
    flex-direction: row-reverse;
    gap: ${sc("--spacing-s")};
    align-self: center;
    justify-content: center;

    /* Button spacing managed via gap instead of margins */
    .sdsm-button {
      margin: 0;
    }

    ${st} {
      flex-wrap: wrap;
      width: 100%;
    }

    ${so} {
      /* See note above, accounts for content rendered in reverse order. */
      flex-direction: column-reverse;
      align-items: center;
    }
  }
`,dy=(0,E.css)`
  .settings-page-footer {
    padding-top: ${sc("--spacing-xl")};
    display: flex;
    flex-direction: row;
    gap: ${sc("--spacing-s")};
    align-self: center;
    justify-content: center;

    /* Button spacing managed via gap instead of margins */
    .sdsm-button {
      margin: 0;
    }

    ${so} {
      padding-top: unset;
      flex-direction: column;
    }
  }
`,dk=(0,E.css)`
  /* card style. */
  background-color: ${sc("--background-color")};
  color: ${sc("--foreground-color")};
  border-radius: ${sc("--border-radius-m")};
  padding: ${sc("--spacing-m")};
  border: 1px #0003 solid;
  box-shadow: ${sc("--box-shadow-l")};
`,dw=(0,E.css)`
  --padding-horizontal: ${sc("--spacing-xl")};
  --padding-vertical: ${sc("--spacing-l")};

  ${st} {
    --padding-horizontal: ${sc("--spacing-s")};
    --padding-vertical: ${sc("--spacing-xs")};
  }
  --max-height: calc(100vh - var(--padding-vertical));
  /* stylelint-disable-next-line declaration-block-no-duplicate-custom-properties */
  --max-height: calc(100dvh - var(--padding-vertical));

  .sdsm-modal-content {
    ${dk}
    padding: var(--padding-vertical) var(--padding-horizontal);
    width: max-content;

    /* stop-gap solution to prevent any part from being hidden */
    overflow: auto;

    /* prevents overflow of this container. */
    max-height: 100%;
    max-width: 100%;
  }

  .cookie-landing-screen,
  .cookie-settings-screen {
    max-height: calc(var(--max-height) - calc(2 * calc(1px + var(--padding-vertical))));

    display: flex;
    flex-direction: column;

    /* Header Styles */
    ${df}

    /* Body Styles */
    .modal-body {
      flex: 1;
      overflow-y: auto;
      min-height: 5em;
      margin-bottom: ${sc("--spacing-xs")};
      padding-right: ${sc("--spacing-xs")};

      *[dir='rtl'] & {
        padding-right: unset;
        padding-left: ${sc("--spacing-xs")};
      }

      .cookie-title {
        ${lm}

        margin-bottom: ${sc("--spacing-m")};
      }

      h3,
      b {
        font-weight: 500;
      }

      ${dv}

      ${dm}
    }

    /* Footer Styles */
    ${db}
  }

  /* Dark Mode Styles */
  ${dg}
`;(0,E.css)`
  /* Body Styles */
  .settings-page-body {
    text-align: left;

    .cookie-title {
      ${lm}

      margin-bottom: ${sc("--spacing-m")};
    }

    ${dv}

    ${dm}
  }

  /* Footer Styles */
  ${dy}

  /* Dark Mode Styles */
  ${dg}
`;var dx=ew(({backgroundType:e,className:t,children:n,isDisplayed:o,portalRoot:r})=>{let i=lA(),{width:a}=lt(),[s,l]=(0,b.useState)(a8.Middle);return(0,b.useEffect)(()=>{let e=a8.Middle;"Mobile"===i&&(a??Number.POSITIVE_INFINITY)<=480&&(e=a8.Bottom),l(e)},[i,a,l]),(0,C.tZ)(uU,{isBlocking:!0,disableBackgroundScroll:!0,verticalAlignment:s,isDisplayed:o,portalRoot:r,className:(0,E.cx)(dp,s7(e),dw,t),contentClassName:(0,E.cx)(dk),children:n})},"SdsmCookieModal"),dS={buttonComponent:({text:e,isDisabled:t,isPrimary:n,onClick:o})=>{let r=n?ut.Primary:void 0;return(0,C.tZ)(un,{disabled:t,onClick:o,size:"Compact",type:r,children:e})},localeDropdownComponent:({currentLocale:e,supportedLocales:t,containerProvider:n,onLocaleChange:o})=>(0,C.tZ)(dd,{currentLocale:e.code,supportedLocales:t,containerProvider:n,onLocaleChange:o}),modalComponent:({backgroundType:e,children:t,isDisplayed:n,portalRoot:o})=>(0,C.tZ)(dx,{backgroundType:e,isDisplayed:n,portalRoot:o,children:t}),sectionComponent:({backgroundType:e,children:t})=>(0,C.tZ)("section",{className:(0,E.cx)(dk,s7(e)),children:t}),toggleComponent:({id:e,isChecked:t,onToggle:n})=>{let o=ew(()=>void 0,"noop");return(0,C.tZ)(uY,{id:e,isChecked:t,onToggle:n??o})}},dC=ew(({portalRoot:e,cookieDomain:t,supportedLocales:n,GTMID:o="",GAID:r="",nonce:i,plugins:a,onLocaleChange:s,onComplete:l,preferencesAcceptedCallback:c,performanceAnalyticsCallback:u,marketingAnalyticsCallback:d,backgroundType:h,forceVisible:p})=>{(0,b.useEffect)(()=>{aQ(r,!1)},[r]);let f=(0,b.useContext)(aq),{globalApolloClient:m,currentLocale:g,isPreview:v,isSSR:y,onEvent:k,onError:w}=f;t??(t=f.hostname),n??(n=f.supportedLocales??aA),s??(s=f.onLocaleChange);let x={...dS,isSSR:y,isPreview:v,currentLocale:g,client:m},S=ew(({label:e})=>k?.({component:"CookieModal",label:e,action:"Click"}),"logEvent"),F={supportedLocales:n,cookieDomain:t,portalRoot:e,backgroundType:h,onLocaleChange:s,onComplete:ew(e=>{let{cookieAcceptance:t,userLocation:n}=e;aG({userLocation:n}),t.Performance&&aX(o,r,a,i),t.Preferences&&c?.(),t.Performance&&u?.(),t.Marketing&&d?.(),l?.(e)},"onComplete"),onEvent:S,onError:w,forceVisible:p};return(0,C.tZ)(aR,{...x,children:(0,C.tZ)(aj,{...F})})},"CookieModal"),dF="CookieSettings",d_=ew(({backgroundType:e="White",cookieDomain:t="",onSubmit:n,onEvent:o})=>{let r=(0,b.useContext)(at),{data:i}=ae(ar,r,{client:r.client}),a=w(i?.cookieModalCollection.items),[s,l]=(0,b.useMemo)(()=>a?.cookieCategoriesCollection?.items?[a.cookieCategoriesCollection.items.map(e=>{let{categoryCookieName:t,displayMode:n,enableToggle:o,isEssential:r}=e;return{categoryCookieName:t,displayMode:n,enableToggle:o,isEssential:r}}),am(a.cookieCategoriesCollection.items)]:[],[a]),[c]=(0,b.useState)(ac(t)),[u,d]=(0,b.useState)("Unknown"),[h,p]=(0,b.useState)({}),f=ew((e,t)=>{let n=k(h);n[e]=t,p(n),g("Enabled")},"updateCategoryState"),[m,g]=(0,b.useState)("Disabled"),y=ew(()=>{g("Saved"),setTimeout(()=>{g("Disabled")},3e3)},"updateSaveButton"),[x,S]=(0,b.useState)(),C=(0,b.useCallback)(e=>{l&&s&&(au(c,e),av(e,l))},[c,l,s]);(0,b.useEffect)(()=>{let e=window.location.hostname;S(new aa(v.PARTITION.COOKIE_MODAL_COMPONENTS,e))},[S]),(0,b.useEffect)(()=>{if(!s)return;let e=setInterval(()=>{let t=ad(s);t["sc-cookies-accepted"]&&(p(t),clearInterval(e))},1e3);return()=>clearInterval(e)},[s,p]),(0,b.useEffect)(()=>{x&&ew(async()=>{try{let e=await ak();d(e)}catch(t){let e=eC(t);x.logError(dF,"userLocation",e)}},"fetchUserRegion")()},[d,x]);let F=ew(()=>{if(!s)return;let e=af(s,h);C(e),y(),_("save_changes",e),n?.({userLocation:u,cookieAcceptance:e})},"acceptSelected"),_=ew((e,t)=>{o?.({component:dF,action:"Click",label:e});let n={preferences:`${t.Preferences??!1}`,performance:`${t.Performance??!1}`,marketing:`${t.Marketing??!1}`,userLocation:u};x?.logMetric(`${e}_clicks_settings`,n)},"logUserAction");if(!a)return null;let N={isDisabled:"Disabled"===m,isPrimary:"Saved"!==m,text:"Saved"===m?a.changesSavedText:a.saveChangesText,onClick:F};return(0,O.jsxs)(aC,{backgroundType:e,children:[(0,O.jsx)("div",{"data-testid":"mwp-cookie-settings-page-body",className:"settings-page-body",children:(0,O.jsx)(az,{title:a.settingsScreenTitle,cookieCategories:a.cookieCategoriesCollection.items,categoriesState:h,updateCategoriesState:f,activeToggleLabel:a.toggleEnabledText,inactiveToggleLabel:a.toggleDisabledText})}),(0,O.jsx)("div",{"data-test-id":"mwp-cookie-settings-page-footer",className:"settings-page-footer",children:(0,O.jsx)(aw,{...N})})]})},"CookieSettings");d_.displayName=dF,aN(d_,v.PARTITION.COOKIE_MODAL_COMPONENTS);var dN=new Set(["slow-2g","2g","3g"]);function d$(){let e=(0,b.useContext)(a4),t=e.getCachedHighEntropyHints()?.connection,n=e.getLowEntropyHints().saveData||!!(t&&dN.has(t)),o=e.getLowEntropyHints();return dc({isSlowConnection:n,supportsAvif:a1(o),supportsWebP:a2(o)})}ew(d$,"useContentfulImages");var dE=[{kind:"FragmentDefinition",name:{kind:"Name",value:"AssetAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"Asset"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"description"}},{kind:"Field",name:{kind:"Name",value:"url"}},{kind:"Field",name:{kind:"Name",value:"contentType"}}]}}],dO=[{kind:"FragmentDefinition",name:{kind:"Name",value:"ImageAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"Image"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"media"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"AssetAll"}}]}},{kind:"Field",name:{kind:"Name",value:"wrap"}}]}}],dD=[{kind:"FragmentDefinition",name:{kind:"Name",value:"FooterItemV3All"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterItemV3"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"url"}},{kind:"Field",name:{kind:"Name",value:"analytics"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"label"}}]}},{kind:"Field",name:{kind:"Name",value:"hideOnDomains"}}]}}],dT=[{kind:"FragmentDefinition",name:{kind:"Name",value:"FooterLocaleDropdownAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterLocaleDropdown"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"hideOnDomains"}}]}}],dz=[{kind:"FragmentDefinition",name:{kind:"Name",value:"FooterCookiesSettingsLinkAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterCookiesSettingsLink"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"hideOnDomains"}},{kind:"Field",name:{kind:"Name",value:"analytics"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"label"}}]}}]}}],dM=[{kind:"FragmentDefinition",name:{kind:"Name",value:"FooterGroupAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterGroup"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"groupKey"}},{kind:"Field",name:{kind:"Name",value:"itemsCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"10"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"__typename"}},{kind:"InlineFragment",typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterItemV3"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"FooterItemV3All"}}]}},{kind:"InlineFragment",typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterLocaleDropdown"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"FooterLocaleDropdownAll"}}]}},{kind:"InlineFragment",typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterCookiesSettingsLink"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"FooterCookiesSettingsLinkAll"}}]}}]}}]}},{kind:"Field",name:{kind:"Name",value:"analytics"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"label"}}]}},{kind:"Field",name:{kind:"Name",value:"hideOnDomains"}}]}}],dI=[{kind:"FragmentDefinition",name:{kind:"Name",value:"FooterV3All"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterV3"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"logo"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"ImageAll"}}]}},{kind:"Field",name:{kind:"Name",value:"url"}},{kind:"Field",name:{kind:"Name",value:"hideLogoOnDomains"}},{kind:"Field",name:{kind:"Name",value:"columnsCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"10"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"InlineFragment",typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterGroup"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"FooterGroupAll"}}]}}]}}]}},{kind:"Field",name:{kind:"Name",value:"barCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"10"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"InlineFragment",typeCondition:{kind:"NamedType",name:{kind:"Name",value:"FooterGroup"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"FooterGroupAll"}}]}}]}}]}}]}}],dP=[{kind:"FragmentDefinition",name:{kind:"Name",value:"ButtonAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"Button"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"json"}}]}},{kind:"Field",name:{kind:"Name",value:"url"}},{kind:"Field",name:{kind:"Name",value:"size"}},{kind:"Field",name:{kind:"Name",value:"theme"}},{kind:"Field",name:{kind:"Name",value:"buttonType"}},{kind:"Field",name:{kind:"Name",value:"image"},selectionSet:{kind:"SelectionSet",selections:[{kind:"InlineFragment",typeCondition:{kind:"NamedType",name:{kind:"Name",value:"Image"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"ImageAll"}}]}}]}},{kind:"Field",name:{kind:"Name",value:"iconName"}}]}}];[...dP,...dO,...dE],[...dI,...dO,...dE,...dM,...dD,...dT,...dz],[...dD],[...dI,...dO,...dE,...dM,...dD,...dT,...dz];var dj={kind:"Document",definitions:[{kind:"OperationDefinition",operation:"query",name:{kind:"Name",value:"GlobalNavConfigCollectionQuery"},variableDefinitions:[{kind:"VariableDefinition",variable:{kind:"Variable",name:{kind:"Name",value:"preview"}},type:{kind:"NonNullType",type:{kind:"NamedType",name:{kind:"Name",value:"Boolean"}}}},{kind:"VariableDefinition",variable:{kind:"Variable",name:{kind:"Name",value:"locale"}},type:{kind:"NonNullType",type:{kind:"NamedType",name:{kind:"Name",value:"String"}}}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"globalNavConfigCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"preview"},value:{kind:"Variable",name:{kind:"Name",value:"preview"}}},{kind:"Argument",name:{kind:"Name",value:"locale"},value:{kind:"Variable",name:{kind:"Name",value:"locale"}}},{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"1"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"globalNavLabel"}},{kind:"Field",name:{kind:"Name",value:"globalNavGroupsCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"20"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"GlobalNavAll"}}]}}]}}]}}]}}]}},{kind:"FragmentDefinition",name:{kind:"Name",value:"GlobalNavAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"GlobalNav"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"groupKey"}},{kind:"Field",name:{kind:"Name",value:"primaryHostnameRegex"}},{kind:"Field",name:{kind:"Name",value:"hideHostnameRegex"}},{kind:"Field",name:{kind:"Name",value:"highlight"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"GlobalNavHighlightAll"}}]}},{kind:"Field",name:{kind:"Name",value:"itemsCollection"},arguments:[{kind:"Argument",name:{kind:"Name",value:"limit"},value:{kind:"IntValue",value:"20"}}],selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"items"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"GlobalNavItemAll"}}]}}]}}]}},{kind:"FragmentDefinition",name:{kind:"Name",value:"GlobalNavHighlightAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"GlobalNavHighlight"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"cta"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"CallToActionAll"}}]}},{kind:"Field",name:{kind:"Name",value:"background"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"AssetAll"}}]}},{kind:"Field",name:{kind:"Name",value:"cardTitleV2"}},{kind:"Field",name:{kind:"Name",value:"cardBodyV2"}}]}},{kind:"FragmentDefinition",name:{kind:"Name",value:"CallToActionAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"CallToAction"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"analytics"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"AnalyticsAll"}}]}},{kind:"Field",name:{kind:"Name",value:"presentation"},selectionSet:{kind:"SelectionSet",selections:[{kind:"InlineFragment",typeCondition:{kind:"NamedType",name:{kind:"Name",value:"Button"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"ButtonAll"}}]}}]}},{kind:"Field",name:{kind:"Name",value:"url"}}]}},{kind:"FragmentDefinition",name:{kind:"Name",value:"AnalyticsAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"Analytics"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"label"}}]}},...dP,...dO,...dE,{kind:"FragmentDefinition",name:{kind:"Name",value:"GlobalNavItemAll"},typeCondition:{kind:"NamedType",name:{kind:"Name",value:"GlobalNavItem"}},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"sys"},selectionSet:{kind:"SelectionSet",selections:[{kind:"Field",name:{kind:"Name",value:"id"}}]}},{kind:"Field",name:{kind:"Name",value:"title"}},{kind:"Field",name:{kind:"Name",value:"url"}},{kind:"Field",name:{kind:"Name",value:"hideHostnameRegex"}},{kind:"Field",name:{kind:"Name",value:"analytics"},selectionSet:{kind:"SelectionSet",selections:[{kind:"FragmentSpread",name:{kind:"Name",value:"AnalyticsAll"}}]}}]}}]};[...dE],(0,E.css)`
  width: 100%;

  ${sn} {
    width: auto;
  }
`,(0,E.css)`
  padding-block-end: ${8}px;

  ${sn} {
    padding-block-end: 0;
  }
`;var dR=(0,E.css)`
  ${st} {
    width: 100%;
    .sdsm-dropdown,
    .sdsm-dropdown-button {
      width: 100%;
    }
  }
`,dL=(0,E.css)`
  display: flex;
  flex-direction: column;
  gap: ${sc("--spacing-s")};
`,dA=(0,E.css)`
  flex-direction: row;
  align-items: center;

  ${st} {
    flex-direction: column;
    align-items: flex-start;
  }
`,dV=(0,E.css)`
  font-size: 14px;
  font-weight: 600;
  line-height: 16px;
  color: ${sc("--foreground-color")};
`;ew(({title:e,currentOrientation:t})=>(0,C.BX)("div",{className:(0,E.cx)(dL,{[dA]:"Horizontal"===t}),children:[e&&(0,C.tZ)("label",{className:dV,children:e}),(0,C.tZ)(dd,{className:dR})]}),"FooterLocaleDropdown").displayName="FooterLocaleDropdown";var dq=((c=dq||{}).FULL_FOOTER="Full Footer",c.NO_FOOTER="No Footer",c.SIMPLE_FOOTER="Simple Footer",c),dZ=ew(({document:e})=>"string"==typeof e?(0,C.tZ)(C.HY,{children:e}):(0,C.tZ)(C.HY,{children:(0,eg.a)(e)}),"PlainRichText");dZ.displayName="PlainRichText";var dB=ew(e=>{let{onNavigate:t,onEvent:n}=(0,b.useContext)(aq),{url:o,buttonType:r,theme:i,title:a,size:s,image:l,iconName:c}=e,u=ew(()=>{n&&n({action:eS.Click,component:"Button",url:o,label:e.analyticsLabel}),z(o)||(t?t(o):window.open(o))},"onClick");return(0,C.tZ)(un,{link:o,type:r??i,onClick:u,size:s,image:l?.media?{...l?.media,width:24,height:24}:void 0,iconName:c,children:a?.json&&(0,C.tZ)(dZ,{document:a.json})})},"Button");dB.displayName="Button";var dQ=ew(e=>{let{onError:t}=(0,b.useContext)(aq),{presentation:n,analytics:o,url:r}=e;return(r&&t?.(Error('The "url" field on CallToAction is no longer supported')),"Button"===n.__typename)?(0,C.tZ)(dB,{analyticsLabel:o?.label,...n}):(t?.(Error(`Unable to render CTA presentation "${n.__typename}"`)),null)},"CallToAction");dQ.displayName="CallToAction";var dH=ew(({triggered:e,ctaProps:t})=>{let{onEvent:n,onNavigate:o}=(0,b.useContext)(aq),{presentation:r,url:i,analytics:a}=t??{},s=r?.__typename==="Button";(0,b.useEffect)(()=>{if(!e||!s)return;let t=i??r?.url;n&&n({action:eS.Click,component:"VirtualCallToAction",label:a?.label,url:t}),o?o({url:t}):window&&(window.location.href=t)},[e,i,a?.label,n,o,s,r?.url])},"useActivation"),dG=ew(e=>!!e&&e.startsWith("video"),"isVideoUrl"),dU=ew(e=>!!e&&e.startsWith("image"),"isImageUrl"),dW=ew(e=>{let{onError:t}=(0,b.useContext)(aq),{contentType:n,url:o,description:r,sizedSrcSets:i}=e,{getImageSources:a}=d$();if(dU(n)){let e=a(o);return i&&(e=a(o,{size:i})),(0,C.tZ)(uq,{sourceType:n,marginBottom:!1,imgSrcs:e,altText:r})}return dG(n)?(0,C.tZ)(uq,{sourceType:n,marginBottom:!1,videoSource:o,showVideoControls:!0,maxHeight:600,altText:r}):(t?.(`Unsupported Mdea type ${n}`),null)},"Media"),dK={sizeToUrl:[480,960,1440,1920,2560].map(e=>({size:`${e}w`,settings:{quality:70,width:e}})),sizes:"(max-width: 768px) 100vw, calc(100vw - 452px)"},dY=ew(e=>{let[t,n]=(0,b.useState)(0);dH({triggered:t,ctaProps:e?.cta});let{cta:o,background:r,cardTitleV2:i,cardBodyV2:a}=e,s=ew(()=>{n(t+1)},"onActivate");return(0,C.tZ)(cN,{cardTitleV2:i,cardBodyV2:a,background:(0,C.tZ)(dW,{...r,sizedSrcSets:dK}),callToAction:(0,C.tZ)(dQ,{...o}),onActivate:s})},"GlobalNavHighlight");dY.displayName="GlobalNavHighlight";var dX=ew((e,t)=>!!e&&new RegExp(e).test(t),"isHiddenNavItem"),dJ=ew(({hideHostnameRegex:e,title:t,url:n,analytics:o})=>{let{onEvent:r}=(0,b.useContext)(aq),i=ew(()=>{r&&r({action:eS.Click,component:"GlobalNavItem",label:o?.label,url:n})},"onClick");return dX(e,window.location.hostname)?null:(0,C.tZ)(cM,{title:t,href:n,showExternalIcon:!1,onClick:i})},"GlobalNavItem");dJ.displayName="GlobalNavItem";var d0=ew(({title:e,groupKey:t,itemsCollection:n,highlight:o})=>{let r=(0,C.tZ)(dY,{...o});return(0,C.tZ)(ch,{navGroupKey:t,title:e,mobileHighlight:r,isExpandable:!0,children:n?.items.map((e,t)=>C.tZ(dJ,{...e},`nav-item-${t}`))})},"GlobalNavGroup");function d1(e,t){return[...e.filter(t),...ev(e,t)]}d0.displayName="GlobalNavGroup",ew(d1,"sortNavGroups");var d2=ew((e,t)=>e.filter(e=>!e.hideHostnameRegex||new RegExp(e.hideHostnameRegex).test(t)),"filterNavGroups"),d3=ew(({navGroups:e})=>{let{hostname:t}=(0,b.useContext)(aq),n=ew(e=>!!e.primaryHostnameRegex&&new RegExp(e.primaryHostnameRegex).test(t),"isPromoted"),o=d1(d2(e,t),n);return(0,C.tZ)(C.HY,{children:o.map(e=>(0,C.tZ)(d0,{...e},e.groupKey))})},"GlobalNavGroupCollection");d3.displayName="GlobalNavGroupCollection";var d5=ew(e=>{let t=new Map;return e.navGroups.filter(e=>!!e.highlight).forEach(e=>{t.set(e.groupKey,(0,C.tZ)(dY,{...e.highlight}))}),(0,C.tZ)(c$,{highlights:t})},"GlobalNavHighlightCollection");d5.displayName="GlobalNavHighlightCollection";var d4=ew(({backgroundColor:e,localNavMobile:t,localNavMobileFooter:n,navGroups:o,globalNavHeading:r,showMobileGlobalLinks:i})=>{let{toggleExpanded:a}=(0,b.useContext)(lY);return(0,C.tZ)(cY,{localNavMobile:t,localNavMobileFooter:n,onNavClose:a??(()=>null),backgroundColor:e,globalNavHeading:r,highlight:(0,C.tZ)(d5,{navGroups:o}),globalNav:(0,C.tZ)(d3,{navGroups:o}),showMobileGlobalLinks:i})},"GlobalNavGroupScreen");d4.displayName="GlobalNavGroupScreen";var d6=ew((e,t=!1,n)=>{let{onError:o}=(0,b.useContext)(aq);(0,b.useEffect)(()=>{if(!(!document||!e||n?.skipPreload?.())){if(t){let t=e.map(e=>{let t=document.createElement("link");return t.rel="preload",t.as=dG(e.contentType)?"video":"image",t.href=e.url,t});document.head.append(...t)}else for(let t of e)if(dG(t.contentType)){let e=document.createElement("video"),n=document.createElement("source");n.type=t.contentType,n.src=t.url,e.appendChild(n),e.load()}else if(dU(t.contentType)){let e=document.createElement("picture"),o=document.createElement("source"),r=document.createElement("source"),i=document.createElement("img");if(t.url.endsWith(".svg"))i.src=t.url;else if(n?.sizedSrcSets){let{sizedSrcSets:a}=n;o.srcset=dl(t.url,{size:a},"avif"),o.type="image/avif",r.srcset=dl(t.url,{size:a},"webp"),r.type="image/webp",a.sizes&&(o.sizes=a.sizes,r.sizes=a.sizes,i.sizes=a.sizes),e.appendChild(o),e.appendChild(r),e.appendChild(i),i.srcset=dl(t.url,{size:a},"jpg")}else o.srcset=ds({imageUrl:t.url,settings:{format:"avif"}}),o.type="image/avif",r.srcset=ds({imageUrl:t.url,settings:{format:"webp"}}),r.type="image/webp",e.appendChild(o),e.appendChild(r),e.appendChild(i),i.src=t.url}else o?.(`Cannot preload assets of type ${t.contentType}`)}},[e,t,o,n])},"useMediaPreload"),d8=a9.Black;lY.displayName="GlobalHeaderContextSDSM";var d9={sizedSrcSets:dK,skipPreload:()=>!window||window.innerWidth<=768};function d7(e){var t=e||{},n=t.initial,o=void 0===n?300:n,r=t.jitter,i=void 0===r||r,a=t.max,s=void 0===a?1/0:a,l=i?o:o/2;return ew(function(e){var t=Math.min(s,l*Math.pow(2,e));return i&&(t=Math.random()*t),t},"delayFunction")}function he(e){var t=e||{},n=t.retryIf,o=t.max,r=void 0===o?5:o;return ew(function(e,t,o){return!(e>=r)&&(n?n(o,t):!!o)},"retryFunction")}ew(({backgroundColor:e,className:t,defaultGroupKey:n,siteName:o,displayed:r,trackingSiteName:i,cta:a,ctaItems:s,localNavDesktop:l,localNavItems:c,logo:u,logoProps:d,localNavMobile:h,localNavMobileFooter:p,onToggleExpanded:f,search:m,searchOpen:g,endChildrenDesktopClassName:v,endChildrenMobileClassName:y,showGlobalLinks:k=!0,showNavScreen:x,pathname:S,headerNavigationTree:F})=>{let _=(0,b.useContext)(aq),{onEvent:N}=_,{data:$}=ae(dj,{currentLocale:_.currentLocale??aL,isPreview:_.isPreview??!1,isSSR:_.isSSR??!1},{client:_.globalApolloClient}),E=w($?.globalNavConfigCollection.items),O=(0,b.useMemo)(()=>E?.globalNavGroupsCollection.items??[],[E?.globalNavGroupsCollection.items]),D=a6((0,b.useContext)(a4));d6((0,b.useMemo)(()=>k?O.map(e=>e.highlight?.background).filter(e=>!!e):[],[O,k]),!1,d9);let T=ew(e=>{N&&N({component:"GlobalHeader",action:eS.Click,label:e?"expand":"collapse"}),f?.(e)},"onToggleExpandedWithLogging"),z=a??(0,C.tZ)(C.HY,{children:b.Children.toArray(s?.map((e,t)=>C.tZ(un,{link:e.url,type:e.type,size:"Compact",children:e.title},t)))}),M=g?m:(0,C.BX)(C.HY,{children:[z,m]}),I=d?(0,C.tZ)(u3,{backgroundColor:e??d8,...d}):null,P=h??(0,C.tZ)(C.HY,{children:c?.map((e,t)=>C.tZ(cM,{title:e.title,href:"url"in e?e.url:void 0,children:"items"in e?e.items.map((e,n)=>C.tZ(cM,{title:e.title,href:"url"in e?e.url:void 0,children:"items"in e?e.items.map((e,o)=>C.tZ(cM,{title:e.title,href:"url"in e?e.url:void 0},`${t}.${n}.${o}`)):void 0},`${t}.${n}`)):void 0},t))}),j=l??(0,C.tZ)(C.HY,{children:c?.map((e,t)=>C.tZ(da,{id:String(t),title:e.title,url:"url"in e?e.url:void 0,children:"items"in e?e.items.map((e,n)=>C.tZ(da,{id:`${t}.${n}`,title:e.title,url:"url"in e?e.url:void 0,children:"items"in e?e.items.map((e,o)=>C.tZ(da,{id:`${t}.${n}.${o}`,title:e.title,url:"url"in e?e.url:void 0},`${t}.${n}.${o}`)):void 0},`${t}.${n}`)):void 0},t))});return(0,C.tZ)(cn,{className:t,siteName:o,backgroundColor:e??d8,displayed:r,logo:!(g&&D)&&(u??I),onToggleExpanded:T,cta:M,localNavDesktop:j,defaultGroupKey:n??"snapchat",stayOpenInvariant:S,trackingSiteName:i??_.hostname,endChildrenClassName:g?D?y:v:void 0,showGlobalLinks:k,showNavScreen:x,isUrlCurrent:_.isUrlCurrent,headerNavigationTree:F,children:E&&(0,C.tZ)(d4,{backgroundColor:e??d8,localNavMobile:P,localNavMobileFooter:p,globalNavHeading:E?.globalNavLabel,navGroups:O,showMobileGlobalLinks:k})})},"GlobalHeader").displayName="GlobalHeader",ew(d7,"buildDelayFunction"),ew(he,"buildRetryFunction");var ht=function(){function e(e,t,n,o){var r=this;this.operation=e,this.nextLink=t,this.delayFor=n,this.retryIf=o,this.retryCount=0,this.values=[],this.complete=!1,this.canceled=!1,this.observers=[],this.currentSubscription=null,this.onNext=function(e){r.values.push(e);for(var t=0,n=r.observers;t<n.length;t++){var o=n[t];o&&o.next(e)}},this.onComplete=function(){r.complete=!0;for(var e=0,t=r.observers;e<t.length;e++){var n=t[e];n&&n.complete()}},this.onError=function(e){return eM(r,void 0,void 0,function(){var t,n,o;return eI(this,function(r){switch(r.label){case 0:return this.retryCount+=1,[4,this.retryIf(this.retryCount,this.operation,e)];case 1:if(r.sent())return this.scheduleRetry(this.delayFor(this.retryCount,this.operation,e)),[2];for(t=0,this.error=e,n=this.observers;t<n.length;t++)(o=n[t])&&o.error(e);return[2]}})})}}return ew(e,"RetryableOperation"),e.prototype.subscribe=function(e){if(this.canceled)throw Error("Subscribing to a retryable link that was canceled is not supported");this.observers.push(e);for(var t=0,n=this.values;t<n.length;t++){var o=n[t];e.next(o)}this.complete?e.complete():this.error&&e.error(this.error)},e.prototype.unsubscribe=function(e){var t=this.observers.indexOf(e);if(t<0)throw Error("RetryLink BUG! Attempting to unsubscribe unknown observer!");this.observers[t]=null,this.observers.every(function(e){return null===e})&&this.cancel()},e.prototype.start=function(){this.currentSubscription||this.try()},e.prototype.cancel=function(){this.currentSubscription&&this.currentSubscription.unsubscribe(),clearTimeout(this.timerId),this.timerId=void 0,this.currentSubscription=null,this.canceled=!0},e.prototype.try=function(){this.currentSubscription=this.nextLink(this.operation).subscribe({next:this.onNext,error:this.onError,complete:this.onComplete})},e.prototype.scheduleRetry=function(e){var t=this;if(this.timerId)throw Error("RetryLink BUG! Encountered overlapping retries");this.timerId=setTimeout(function(){t.timerId=void 0,t.try()},e)},e}();!function(e){function t(t){var n=e.call(this)||this,o=t||{},r=o.attempts,i=o.delay;return n.delayFor="function"==typeof i?i:d7(i),n.retryIf="function"==typeof r?r:he(r),n}eD(t,e),ew(t,"RetryLink"),t.prototype.request=function(e,t){var n=new ht(e,t,this.delayFor,this.retryIf);return n.start(),new nV(function(e){return n.subscribe(e),function(){n.unsubscribe(e)}})}}(of)}}]);
//# sourceMappingURL=1f362e0b.77178fbe6bc8a0d8.js.map
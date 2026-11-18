package com.alpharobotics.admin;
import org.json.JSONObject;
import java.io.*;
import java.net.*;
import java.nio.charset.StandardCharsets;
import java.util.Map;
final class Net {
 static final class Response{final int status;final String body;final java.util.List<String> cookies;Response(int s,String b,java.util.List<String> cookies){status=s;body=b;this.cookies=cookies;}JSONObject json()throws Exception{return new JSONObject(body);}}
 static Response request(String method,String address,String cookie,Map<String,String> form)throws Exception{
  URL u=new URL(address);if(!"https".equals(u.getProtocol()))throw new IOException("HTTPS 연결만 사용할 수 있습니다.");HttpURLConnection c=(HttpURLConnection)u.openConnection();c.setConnectTimeout(10000);c.setReadTimeout(15000);c.setInstanceFollowRedirects(false);c.setRequestMethod(method);c.setRequestProperty("Accept","application/json");c.setRequestProperty("Cache-Control","no-cache");if(cookie!=null&&!cookie.isEmpty())c.setRequestProperty("Cookie",cookie);
  if(form!=null){StringBuilder b=new StringBuilder();for(Map.Entry<String,String> e:form.entrySet()){if(b.length()>0)b.append('&');b.append(URLEncoder.encode(e.getKey(),"UTF-8")).append('=').append(URLEncoder.encode(e.getValue(),"UTF-8"));}byte[] raw=b.toString().getBytes(StandardCharsets.UTF_8);c.setDoOutput(true);c.setRequestProperty("Content-Type","application/x-www-form-urlencoded");c.setFixedLengthStreamingMode(raw.length);try(OutputStream out=c.getOutputStream()){out.write(raw);}}
  int status=c.getResponseCode();InputStream in=status<400?c.getInputStream():c.getErrorStream();ByteArrayOutputStream b=new ByteArrayOutputStream();if(in!=null)try(InputStream src=in){byte[] chunk=new byte[8192];int n;while((n=src.read(chunk))!=-1){if(b.size()+n>1048576)throw new IOException("응답이 너무 큽니다.");b.write(chunk,0,n);}}java.util.List<String> cookies=new java.util.ArrayList<>();for(Map.Entry<String,java.util.List<String>> h:c.getHeaderFields().entrySet())if(h.getKey()!=null&&h.getKey().equalsIgnoreCase("Set-Cookie"))cookies.addAll(h.getValue());c.disconnect();return new Response(status,b.toString("UTF-8"),cookies);
 }
 static String endpoint()throws Exception{String address="https://raw.githubusercontent.com/swsgood79-beep/Tool/main/qr-endpoint.json";try{Response r=request("GET","https://api.github.com/repos/swsgood79-beep/Tool/commits?path=qr-endpoint.json&per_page=1&t="+System.currentTimeMillis(),null,null);if(r.status==200){org.json.JSONArray a=new org.json.JSONArray(r.body);String sha=a.getJSONObject(0).optString("sha");if(sha.matches("[a-f0-9]{40}"))address="https://raw.githubusercontent.com/swsgood79-beep/Tool/"+sha+"/qr-endpoint.json";}}catch(Exception ignored){}
  Response r=request("GET",address+"?t="+System.currentTimeMillis(),null,null);if(r.status!=200)throw new IOException("회사 접속 정보를 읽지 못했습니다.");JSONObject j=r.json();if(!"wanryotalk".equals(j.optString("application"))||j.optInt("version")!=1||!j.optBoolean("online"))throw new IOException("회사 PC의 인터넷 연결이 꺼져 있습니다.");String v=UrlPolicy.tunnel(j.optString("endpoint"));if(v==null)throw new IOException("회사 접속 주소를 확인해 주세요.");return v;
 }
}

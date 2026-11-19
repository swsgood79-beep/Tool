package com.alpharobotics.admin;
import android.content.Context;
import android.os.Build;
import android.webkit.CookieManager;
import java.util.HashMap;
import java.util.Map;
import org.json.JSONObject;
final class PersistentLogin {
 static void applyCookies(String base,Net.Response r)throws Exception{
  if(r.cookies.isEmpty())throw new java.io.IOException("로그인 쿠키를 받지 못했습니다.");
  java.util.concurrent.CountDownLatch done=new java.util.concurrent.CountDownLatch(r.cookies.size());
  java.util.concurrent.atomic.AtomicBoolean accepted=new java.util.concurrent.atomic.AtomicBoolean(true);
  new android.os.Handler(android.os.Looper.getMainLooper()).post(()->{for(String cookie:r.cookies)CookieManager.getInstance().setCookie(base,cookie,ok->{if(!ok)accepted.set(false);done.countDown();});});
  if(!done.await(10,java.util.concurrent.TimeUnit.SECONDS)||!accepted.get())throw new java.io.IOException("앱 로그인 정보를 적용하지 못했습니다.");
  CookieManager.getInstance().flush();
 }
 static void forget(Store s){s.put("login_grant","");s.putSecret("login_secret","");s.put("csrf_hint","");s.put("login_account","");}
 static void clearCookie(Store s,String base){String name=s.get("cookie_name");if(!name.isEmpty()&&UrlPolicy.tunnel(base)!=null){CookieManager.getInstance().setCookie(base,name+"=; Path=/; Max-Age=0; Secure; HttpOnly");CookieManager.getInstance().flush();}}
 static boolean restore(Context c,String base)throws Exception{Store s=new Store(c);if(s.flag("signed_out")||s.get("login_grant").isEmpty()||s.getSecret("login_secret").isEmpty())return false;Map<String,String> f=new HashMap<>();f.put("grant",s.get("login_grant"));f.put("secret",s.getSecret("login_secret"));f.put("device",s.device());f.put("csrf_hint",s.get("csrf_hint"));Net.Response r=Net.request("POST",base+"/native-admin/session/restore",null,f);if(r.status==200){applyCookies(base,r);s.put("csrf_hint",r.json().optString("csrf"));return true;}if(r.status==403&&"LOGIN_REVOKED".equals(r.json().optString("code"))){forget(s);clearCookie(s,base);}return false;}
 static void enroll(Context c,String base,String cookie,String csrf,String account,boolean valid)throws Exception{Store s=new Store(c);if(valid&&!s.get("login_grant").isEmpty()&&!s.getSecret("login_secret").isEmpty()&&account.equals(s.get("login_account")))return;if(!s.get("login_account").isEmpty()&&!account.equals(s.get("login_account"))){revoke(base,s.get("login_grant"),s.getSecret("login_secret"),s.device());forget(s);}Map<String,String> f=new HashMap<>();f.put("csrf",csrf);f.put("device",s.device());f.put("name",Build.MANUFACTURER+" "+Build.MODEL);Net.Response r=Net.request("POST",base+"/native-admin/session/enroll",cookie,f);if(r.status!=200)throw new java.io.IOException("앱 로그인 유지를 연결하지 못했습니다.");JSONObject j=r.json();if(s.flag("signed_out")){revoke(base,j.getString("grant"),j.getString("secret"),s.device());return;}s.putSecret("login_secret",j.getString("secret"));s.put("login_grant",j.getString("grant"));s.put("login_account",account);s.put("cookie_name",j.getString("cookie_name"));s.put("csrf_hint",csrf);s.put("registered_token","");applyCookies(base,r);}
 static void revoke(String base,String id,String secret,String device){if(id.isEmpty()||secret.isEmpty()||UrlPolicy.tunnel(base)==null)return;try{Map<String,String> f=new HashMap<>();f.put("grant",id);f.put("secret",secret);f.put("device",device);Net.request("POST",base+"/native-admin/session/revoke",null,f);}catch(Exception ignored){}}
}

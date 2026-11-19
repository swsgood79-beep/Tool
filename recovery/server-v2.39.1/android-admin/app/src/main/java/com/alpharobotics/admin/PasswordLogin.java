package com.alpharobotics.admin;
import android.content.Context;
import android.webkit.CookieManager;
import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import org.json.JSONObject;
/** Submits the existing normal login form and CSRF nonce; never stores a password. */
final class PasswordLogin {
 private static void alive(Context c)throws IOException{if(c instanceof android.app.Activity&&(((android.app.Activity)c).isDestroyed()||((android.app.Activity)c).isFinishing()))throw new IOException("로그인이 취소되었습니다.");}
 static synchronized JSONObject login(Context c,String base,String username,String password)throws Exception{
  alive(c);Store s=new Store(c);String oldAccount=s.get("login_account"),oldCsrf=s.get("csrf_hint");
  Net.Response page=Net.request("GET",base+"/login",null,null);String nonce=LoginPolicy.nonce(page.body);
  if(page.status!=200||nonce.isEmpty())throw new IOException("회사 PC의 로그인 화면을 확인할 수 없습니다.");
  alive(c);PersistentLogin.applyCookies(base,page);
  Map<String,String> f=new HashMap<>();f.put("csrf",nonce);f.put("login_mode","account");f.put("username",username);f.put("password",password);
  Net.Response result=Net.request("POST",base+"/login",CookieManager.getInstance().getCookie(base+"/login"),f);
  if(result.status!=303)throw new IOException(result.status==429?"로그인 시도가 많습니다. 잠시 후 다시 시도해 주세요.":"아이디 또는 비밀번호를 확인해 주세요.");
  alive(c);PersistentLogin.applyCookies(base,result);
  String sessionCookie=LoginPolicy.cookies(result.cookies);
  Net.Response proof=Net.request("GET",base+"/native-admin/bootstrap?device="+s.device(),sessionCookie,null);
  if(proof.status!=200)throw new IOException("관리자 로그인 상태를 확인할 수 없습니다.");
  JSONObject j=proof.json();if(j.has("account_username")&&!username.equals(LoginPolicy.username(j.getString("account_username"))))throw new IOException("로그인 계정이 일치하지 않습니다. 다시 시도해 주세요.");s.flag("signed_out",false);
  alive(c);PersistentLogin.enroll(c,base,sessionCookie,j.getString("csrf"),j.getString("account_key"),j.optBoolean("persistent_login"));
  // Keep existing work-form CSRF when the same account reauthenticates, so an
  // interrupted work screen can resume without dropping its unsaved contents.
  if(LoginPolicy.sameAccount(oldAccount,j.getString("account_key"))&&oldCsrf.matches("[A-Za-z0-9_-]{43}")){
   s.put("csrf_hint",oldCsrf);if(PersistentLogin.restore(c,base))j.put("csrf",s.get("csrf_hint"));
  }
  alive(c);Net.Response finalProof=Net.request("GET",base+"/native-admin/bootstrap?device="+s.device(),CookieManager.getInstance().getCookie(base),null);if(finalProof.status!=200)throw new IOException("로그인 연결을 다시 확인해 주세요.");JSONObject current=finalProof.json();if(!LoginPolicy.sameAccount(j.getString("account_key"),current.getString("account_key")))throw new IOException("로그인 계정이 변경되었습니다. 다시 로그인해 주세요.");j=current;
  s.put("fingerprint_username",username);s.put("last_username",username);s.put("login_label",j.optString("account_label",username));
  return j;
 }
}

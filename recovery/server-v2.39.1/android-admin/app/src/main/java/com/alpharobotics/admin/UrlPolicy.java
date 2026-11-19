package com.alpharobotics.admin;
import java.net.URI;
public final class UrlPolicy {
 public static final String ENTRY="https://swsgood79-beep.github.io/Tool/admin/";
 public static String tunnel(String raw){try{URI u=new URI(raw);if(!"https".equals(u.getScheme())||u.getUserInfo()!=null||u.getPort()!=-1||u.getQuery()!=null||u.getFragment()!=null||!"".equals(u.getPath())&&!"/".equals(u.getPath()))return null;String host=u.getHost();if(host==null||!host.matches("[a-z0-9]+(?:-[a-z0-9]+)*\\.trycloudflare\\.com"))return null;String origin="https://"+host;return raw.equals(origin)?origin:null;}catch(Exception e){return null;}}
 public static boolean sameOrigin(String url,String base){try{URI u=new URI(url);return tunnel(base)!=null&&u.getUserInfo()==null&&u.getPort()==-1&&"https".equals(u.getScheme())&&u.getHost().equals(new URI(base).getHost());}catch(Exception e){return false;}}
 public static boolean browserPasskey(String raw,String base){try{URI u=new URI(raw);return sameOrigin(raw,base)&&u.getPath().startsWith("/mobile-admin/passkey")||"https".equals(u.getScheme())&&"swsgood79-beep.github.io".equals(u.getHost())&&"/Tool/admin/".equals(u.getPath())&&u.getQuery()!=null&&u.getQuery().contains("passkey=");}catch(Exception e){return false;}}
 public static String nativeAction(String url){if("alpha-admin://check-update".equals(url))return "update";if("alpha-admin://purchase-alert-toggle".equals(url))return "purchase-toggle";if("alpha-admin://notification-settings".equals(url))return "settings";if("alpha-admin://test-notification".equals(url))return "test";return "";}
 public static String purchasing(long call){return call>0?"/purchasing?call="+call:"/purchasing";}
 public static String inspection(long project,long request){return project>0?"/projects/"+project+"/inspection"+(request>0?"?request="+request:""):"/mobile-admin";}
}

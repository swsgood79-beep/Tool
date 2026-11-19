package com.alpharobotics.admin;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.ClipData;
import android.content.Intent;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.content.pm.Signature;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;
import android.widget.Toast;
import androidx.core.content.FileProvider;
import java.io.*;
import java.net.HttpURLConnection;
import java.net.URL;
import java.security.MessageDigest;
import java.util.function.BooleanSupplier;
import org.json.JSONObject;

/** Optional APK updates: fixed source, SHA-256 + installed signer verification, explicit OS approval. */
final class AppUpdater {
 private static final String MANIFEST="https://raw.githubusercontent.com/swsgood79-beep/Tool/recovery-v2.39.1/recovery/server-v2.39.1/android-admin/update.json";
 private final Activity activity;private final BooleanSupplier active;private final java.util.function.Consumer<Runnable> gate;private volatile boolean checking,downloading;private long checkedAt;private File pending;private boolean waitingPermission;
 AppUpdater(Activity a,BooleanSupplier active,java.util.function.Consumer<Runnable> gate){activity=a;this.active=active;this.gate=gate;}
 private long version(PackageInfo p){return Build.VERSION.SDK_INT>=28?p.getLongVersionCode():p.versionCode;}
 private int flags(){return Build.VERSION.SDK_INT>=28?PackageManager.GET_SIGNING_CERTIFICATES:PackageManager.GET_SIGNATURES;}
 private byte[][] signers(PackageInfo p){Signature[] ss=Build.VERSION.SDK_INT>=28&&p.signingInfo!=null?p.signingInfo.getApkContentsSigners():p.signatures;if(ss==null)return null;byte[][] b=new byte[ss.length][];for(int i=0;i<ss.length;i++)b[i]=ss[i].toByteArray();return b;}
 private void say(String s){activity.runOnUiThread(()->{if(active.getAsBoolean())Toast.makeText(activity,s,Toast.LENGTH_LONG).show();});}
 void check(boolean manual){if(!active.getAsBoolean()||checking||downloading||!manual&&System.currentTimeMillis()-checkedAt<300000)return;checking=true;checkedAt=System.currentTimeMillis();new Thread(()->{try{
  Net.Response r=Net.request("GET",MANIFEST+"?t="+System.currentTimeMillis(),null,null);if(r.status!=200)throw new IOException();JSONObject j=r.json();int code=j.getInt("version_code");String name=j.getString("version_name"),url=j.getString("apk_url"),sha=j.getString("sha256");if(j.optInt("schema")!=1||!activity.getPackageName().equals(j.optString("package_name"))||!UpdatePolicy.allowed(url,name,code,sha))throw new IOException();PackageInfo installed=activity.getPackageManager().getPackageInfo(activity.getPackageName(),flags());if(!UpdatePolicy.newer(code,version(installed))){if(manual)say("최신 버전입니다.");return;}String notes=j.optString("notes");if(notes.length()>500)notes=notes.substring(0,500);final String message="버전 "+name+"\n"+notes+"\n\n업데이트 후에도 로그인 정보는 유지됩니다.";activity.runOnUiThread(()->{if(active.getAsBoolean())new AlertDialog.Builder(activity).setTitle("관리자 앱 업데이트").setMessage(message).setNegativeButton("나중에",null).setPositiveButton("업데이트",(d,n)->gate.accept(()->download(url,sha,code,installed))).show();});
 }catch(Exception e){if(manual)say("업데이트 정보를 확인하지 못했습니다. 인터넷 연결을 확인해 주세요.");}finally{checking=false;}},"alpha-update-check").start();}
 private void download(String url,String expected,int code,PackageInfo installed){if(downloading)return;downloading=true;say("업데이트를 다운로드합니다…");new Thread(()->{File partial=null;try{
  File dir=new File(activity.getCacheDir(),"updates");if(!dir.exists()&&!dir.mkdirs())throw new IOException();partial=new File(dir,"incoming.apk");HttpURLConnection c=(HttpURLConnection)new URL(url).openConnection();c.setInstanceFollowRedirects(false);c.setConnectTimeout(10000);c.setReadTimeout(30000);try{if(c.getResponseCode()!=200)throw new IOException();MessageDigest h=MessageDigest.getInstance("SHA-256");long size=0;try(InputStream in=c.getInputStream();OutputStream out=new FileOutputStream(partial)){byte[] b=new byte[16384];int n;while((n=in.read(b))!=-1){size+=n;if(size>32*1024*1024)throw new IOException();out.write(b,0,n);h.update(b,0,n);}}StringBuilder hex=new StringBuilder();for(byte b:h.digest())hex.append(String.format(java.util.Locale.ROOT,"%02x",b&255));if(!expected.equals(hex.toString()))throw new IOException("integrity");}finally{c.disconnect();}
  PackageInfo apk=activity.getPackageManager().getPackageArchiveInfo(partial.getAbsolutePath(),flags());if(apk==null||!activity.getPackageName().equals(apk.packageName)||version(apk)!=code||!UpdatePolicy.newer(code,version(installed))||!UpdatePolicy.sameSigners(signers(installed),signers(apk)))throw new IOException("signature");File ready=new File(dir,"alpha-admin-update.apk");if(ready.exists()&&!ready.delete()||!partial.renameTo(ready))throw new IOException();partial=null;pending=ready;activity.runOnUiThread(()->{if(active.getAsBoolean())gate.accept(this::install);});
 }catch(Exception e){say("업데이트 파일을 확인하지 못했습니다. 다시 시도해 주세요.");}finally{if(partial!=null)partial.delete();downloading=false;}},"alpha-update-download").start();}
 void resumeInstall(){if(pending==null||!active.getAsBoolean())return;if(waitingPermission){waitingPermission=false;if(!activity.getPackageManager().canRequestPackageInstalls()){say("알파 관리자의 앱 설치 허용이 필요합니다.");return;}}gate.accept(this::install);}
 private void install(){if(pending==null||!pending.exists()||!active.getAsBoolean())return;if(!activity.getPackageManager().canRequestPackageInstalls()){new AlertDialog.Builder(activity).setTitle("앱 업데이트 설치 허용").setMessage("처음 한 번 ‘이 출처 허용’을 켜 주세요. 돌아오면 업데이트 설치를 계속합니다.").setNegativeButton("나중에",null).setPositiveButton("설정 열기",(d,n)->{waitingPermission=true;try{activity.startActivity(new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,Uri.parse("package:"+activity.getPackageName())));}catch(Exception e){waitingPermission=false;say("휴대폰 설정에서 알파 관리자의 앱 설치를 허용해 주세요.");}}).show();return;}Uri file=FileProvider.getUriForFile(activity,activity.getPackageName()+".files",pending);Intent i=new Intent(Intent.ACTION_VIEW).setDataAndType(file,"application/vnd.android.package-archive").addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);i.setClipData(ClipData.newUri(activity.getContentResolver(),"알파 관리자 업데이트",file));try{activity.startActivity(i);pending=null;}catch(Exception e){say("설치 화면을 열지 못했습니다. 휴대폰의 앱 설치 허용을 확인해 주세요.");}}
}

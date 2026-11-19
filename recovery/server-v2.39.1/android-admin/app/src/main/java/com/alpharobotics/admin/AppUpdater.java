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
import android.widget.LinearLayout;
import android.widget.ProgressBar;
import android.widget.TextView;
import android.widget.Toast;
import androidx.core.content.FileProvider;
import org.json.JSONObject;
import java.io.*;
import java.net.HttpURLConnection;
import java.net.URL;
import java.security.MessageDigest;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.BooleanSupplier;

/** Fixed source + digest + installed signer checks. Download consent survives recreation. */
final class AppUpdater {
 private static final String MANIFEST="https://raw.githubusercontent.com/swsgood79-beep/Tool/recovery-v2.39.1/recovery/server-v2.39.1/android-admin/update.json";
 private static final AtomicBoolean TRANSFERRING=new AtomicBoolean(false);
 private static final Object DISK=new Object();
 private static volatile long transferred,total=-1;private static volatile int attempt;
 private final Activity activity;private final BooleanSupplier active;private final java.util.function.Consumer<Runnable> gate;
 private boolean checking,preparing,offering,gateQueued,installerLaunched,waitingPermission,permissionSettingsLaunched,deferred,failureShown,progressHidden;private long checkedAt;
 private Offer current;private AlertDialog progressDialog,prompt;private ProgressBar progress;private TextView progressText;
 private static final class Offer {
  final int code;final String name,url,sha,notes;
  Offer(JSONObject j)throws Exception{code=j.getInt("version_code");name=j.getString("version_name");url=j.getString("apk_url");sha=j.getString("sha256");notes=j.optString("notes");if(j.optInt("schema")!=1||!"com.alpharobotics.admin".equals(j.optString("package_name"))||!UpdatePolicy.allowed(url,name,code,sha))throw new IOException("manifest");}
  JSONObject json(String stage)throws Exception{return new JSONObject().put("schema",1).put("package_name","com.alpharobotics.admin").put("version_code",code).put("version_name",name).put("apk_url",url).put("sha256",sha).put("notes",notes).put("stage",stage);}
 }
 AppUpdater(Activity a,BooleanSupplier active,java.util.function.Consumer<Runnable> gate){activity=a;this.active=active;this.gate=gate;}
 private long version(PackageInfo p){return Build.VERSION.SDK_INT>=28?p.getLongVersionCode():p.versionCode;}
 private int flags(){return Build.VERSION.SDK_INT>=28?PackageManager.GET_SIGNING_CERTIFICATES:PackageManager.GET_SIGNATURES;}
 private byte[][] signers(PackageInfo p){Signature[] ss=Build.VERSION.SDK_INT>=28&&p.signingInfo!=null?p.signingInfo.getApkContentsSigners():p.signatures;if(ss==null)return null;byte[][] b=new byte[ss.length][];for(int i=0;i<ss.length;i++)b[i]=ss[i].toByteArray();return b;}
 private File directory()throws IOException{File d=new File(activity.getCacheDir(),"updates");if(!d.exists()&&!d.mkdirs())throw new IOException("directory");return d;}
 private File ready()throws IOException{return new File(directory(),"alpha-admin-update.apk");}
 private void write(Offer offer,String stage)throws Exception{synchronized(DISK){File d=directory(),tmp=new File(d,"job.tmp"),dest=new File(d,"job.json");byte[] b=offer.json(stage).toString().getBytes(java.nio.charset.StandardCharsets.UTF_8);try(FileOutputStream out=new FileOutputStream(tmp)){out.write(b);out.getFD().sync();}if(!tmp.renameTo(dest))throw new IOException("save-job");}}
 private JSONObject read()throws Exception{synchronized(DISK){File f=new File(directory(),"job.json");if(!f.exists())return null;if(f.length()>16384)throw new IOException("job-size");try(InputStream in=new FileInputStream(f)){ByteArrayOutputStream out=new ByteArrayOutputStream();byte[] b=new byte[4096];int n;while((n=in.read(b))!=-1){if(out.size()+n>16384)throw new IOException("job-size");out.write(b,0,n);}return new JSONObject(out.toString("UTF-8"));}}}
 private void forget(){try{synchronized(DISK){new File(directory(),"job.json").delete();ready().delete();}}catch(Exception ignored){}current=null;}
 private void ui(Runnable action){activity.runOnUiThread(()->{if(active.getAsBoolean())action.run();});}
 private void say(String s){ui(()->Toast.makeText(activity,s,Toast.LENGTH_LONG).show());}
 void check(boolean manual){
  if(!active.getAsBoolean())return;
  if(TRANSFERRING.get()){if(manual)progressHidden=false;if(!progressHidden)showProgress();return;}
  if(current!=null||existsJob()){if(manual||!deferred&&!installerLaunched){resume(manual);return;}}
  if(checking||preparing||offering||!manual&&System.currentTimeMillis()-checkedAt<300000)return;
  checking=true;checkedAt=System.currentTimeMillis();new Thread(()->{try{
   Net.Response response=Net.request("GET",MANIFEST+"?t="+System.currentTimeMillis(),null,null);if(response.status!=200)throw new IOException("manifest-http");Offer offer=new Offer(response.json());PackageInfo installed=installed();
   if(!UpdatePolicy.newer(offer.code,version(installed))){if(manual)say("최신 버전입니다.");return;}
   if(!manual&&current!=null&&offer.code<=current.code)return;
   ui(()->showOffer(offer));
  }catch(Exception e){if(manual)say("업데이트 정보를 확인하지 못했습니다. 인터넷 연결을 확인해 주세요.");}
  finally{activity.runOnUiThread(()->checking=false);}},"alpha-update-check").start();
 }
 private boolean existsJob(){try{return new File(directory(),"job.json").exists();}catch(Exception e){return false;}}
 private PackageInfo installed()throws Exception{return activity.getPackageManager().getPackageInfo(activity.getPackageName(),flags());}
 private void showOffer(Offer offer){if(offering||!active.getAsBoolean())return;offering=true;String notes=offer.notes.length()>500?offer.notes.substring(0,500):offer.notes;
  prompt=new AlertDialog.Builder(activity).setTitle("앱 업데이트").setMessage("버전 "+offer.name+"\n"+notes+"\n\n업데이트 후에도 로그인 정보는 유지됩니다.")
   .setNegativeButton("나중에",null).setPositiveButton("업데이트",(d,n)->start(offer))
   .setOnDismissListener(d->offering=false).show();
 }
 // Starting a download never navigates or discards entered work: only installation is gated.
 private void start(Offer offer){if(!active.getAsBoolean()||!TRANSFERRING.compareAndSet(false,true))return;current=offer;deferred=false;failureShown=false;progressHidden=false;installerLaunched=false;attempt=1;transferred=0;total=-1;showProgress();
  new Thread(()->{File partial=null;boolean success=false;try{
   write(offer,"requested");partial=File.createTempFile("incoming-",".apk",directory());
   UpdateTransfer.download(offer.url,offer.sha,partial,address->(HttpURLConnection)new URL(address).openConnection(),Thread::sleep,(a,b,t)->{attempt=a;transferred=b;total=t;});
   validate(partial,offer);
   synchronized(DISK){File finalFile=ready();if(finalFile.exists()&&!finalFile.delete()||!partial.renameTo(finalFile))throw new IOException("rename");partial=null;write(offer,"ready");}
   success=true;
  }catch(Exception e){try{write(offer,"failed");}catch(Exception ignored){}final boolean verification=e instanceof UpdateTransfer.Failure&&"integrity".equals(e.getMessage())||"verification".equals(e.getMessage());ui(()->showFailure(offer,verification));}
  finally{if(partial!=null)partial.delete();TRANSFERRING.set(false);final boolean done=success;ui(()->{hideProgress();if(done){say("다운로드 완료 · 작업을 마치면 설치를 진행합니다.");prepareInstall();}});}
  },"alpha-update-download").start();
 }
 private void showProgress(){if(!active.getAsBoolean()||progressDialog!=null&&progressDialog.isShowing())return;LinearLayout panel=new LinearLayout(activity);panel.setOrientation(LinearLayout.VERTICAL);int pad=(int)(20*activity.getResources().getDisplayMetrics().density);panel.setPadding(pad,pad,pad,pad);progressText=new TextView(activity);panel.addView(progressText);progress=new ProgressBar(activity,null,android.R.attr.progressBarStyleHorizontal);progress.setMax(100);panel.addView(progress,new LinearLayout.LayoutParams(-1,-2));
  progressDialog=new AlertDialog.Builder(activity).setTitle("업데이트 다운로드").setView(panel).setNegativeButton("화면 닫기",(d,n)->progressHidden=true).setOnCancelListener(d->progressHidden=true).create();progressDialog.show();tickProgress();
 }
 private void tickProgress(){if(progressDialog==null||!progressDialog.isShowing()||!active.getAsBoolean())return;
  progress.setIndeterminate(total<=0);if(total>0)progress.setProgress((int)Math.min(100,transferred*100/total));
  String text=total>0?"다운로드 "+Math.min(100,transferred*100/total)+"%":"서버에 연결하고 있습니다…";if(attempt>1)text="다시 연결 중 ("+attempt+"/3)\n"+text;
  progressText.setText(text+"\n화면을 닫아도 다운로드는 계속됩니다.");
  if(TRANSFERRING.get())progressText.postDelayed(this::tickProgress,300);else{hideProgress();resume(false);}
 }
 private void hideProgress(){if(progressDialog!=null){progressDialog.dismiss();progressDialog=null;}}
 private void showFailure(Offer offer,boolean verification){if(failureShown||!active.getAsBoolean())return;failureShown=true;hideProgress();prompt=new AlertDialog.Builder(activity).setTitle("업데이트 다운로드 실패")
  .setMessage(verification?"파일 검증을 통과하지 못해 설치하지 않았습니다. 다시 다운로드해 주세요.":"업데이트를 다운로드하지 못했습니다. 인터넷 연결을 확인한 뒤 다시 시도해 주세요.")
  .setNegativeButton("나중에",(d,n)->deferred=true).setPositiveButton("다시 시도",(d,n)->{failureShown=false;start(offer);}).setOnCancelListener(d->deferred=true).show();
 }
 void resumeInstall(){progressHidden=false;if(permissionSettingsLaunched){permissionSettingsLaunched=false;waitingPermission=false;if(!activity.getPackageManager().canRequestPackageInstalls()){deferred=true;say("앱 설치 허용이 꺼져 있습니다. 앱 업데이트 확인에서 다시 진행해 주세요.");return;}}resume(false);}
 void dispose(){hideProgress();if(prompt!=null){prompt.dismiss();prompt=null;}}
 private void resume(boolean manual){if(!active.getAsBoolean()||preparing||gateQueued||waitingPermission)return;if(TRANSFERRING.get()){showProgress();return;}
  if(manual){deferred=false;failureShown=false;installerLaunched=false;}else if(deferred||installerLaunched)return;
  preparing=true;new Thread(()->{try{
   JSONObject job=read();if(job==null){current=null;return;}Offer offer=new Offer(job);if(!UpdatePolicy.newer(offer.code,version(installed()))){forget();return;}current=offer;String stage=job.optString("stage");
   if("requested".equals(stage)){ui(()->start(offer));return;}
   if("failed".equals(stage)){ui(()->showFailure(offer,false));return;}
   if("launched".equals(stage)&&!manual){deferred=true;return;}
   validate(ready(),offer);ui(()->prepareInstall());
  }catch(Exception e){Offer offer=current;if(offer!=null)ui(()->showFailure(offer,true));else{forget();say("저장된 업데이트를 확인하지 못했습니다. 업데이트 확인을 다시 눌러 주세요.");}}
  finally{activity.runOnUiThread(()->preparing=false);}},"alpha-update-resume").start();
 }
 private void validate(File file,Offer offer)throws Exception{
  if(!file.exists()||file.length()<=0||file.length()>UpdateTransfer.LIMIT)throw new IOException("verification");MessageDigest digest=MessageDigest.getInstance("SHA-256");try(InputStream in=new FileInputStream(file)){byte[] b=new byte[16384];int n;while((n=in.read(b))!=-1)digest.update(b,0,n);}
  StringBuilder sha=new StringBuilder();for(byte b:digest.digest())sha.append(String.format(java.util.Locale.ROOT,"%02x",b&255));if(!offer.sha.equals(sha.toString()))throw new IOException("verification");
  PackageInfo actual=activity.getPackageManager().getPackageArchiveInfo(file.getAbsolutePath(),flags()),installed=installed();if(actual==null||!activity.getPackageName().equals(actual.packageName)||version(actual)!=offer.code||!UpdatePolicy.newer(offer.code,version(installed))||!UpdatePolicy.sameSigners(signers(installed),signers(actual)))throw new IOException("verification");
 }
 private void prepareInstall(){if(!active.getAsBoolean()||gateQueued||current==null)return;hideProgress();gateQueued=true;gate.accept(()->{gateQueued=false;install();});}
 private void install(){if(!active.getAsBoolean()||current==null||installerLaunched)return;
  if(!activity.getPackageManager().canRequestPackageInstalls()){
   if(waitingPermission)return;waitingPermission=true;
   prompt=new AlertDialog.Builder(activity).setTitle("앱 업데이트 설치 허용").setMessage("처음 한 번 ‘이 출처 허용’을 켜 주세요. 돌아오면 다운로드한 파일로 설치를 계속합니다.")
    .setNegativeButton("나중에",(d,n)->{waitingPermission=false;deferred=true;})
    .setPositiveButton("설정 열기",(d,n)->{permissionSettingsLaunched=true;try{activity.startActivity(new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,Uri.parse("package:"+activity.getPackageName())));}catch(Exception e){permissionSettingsLaunched=false;waitingPermission=false;say("휴대폰 설정에서 알파 관리자의 앱 설치를 허용해 주세요.");}})
    .setOnCancelListener(d->{waitingPermission=false;deferred=true;}).show();return;
  }
  waitingPermission=false;
  try{
   File file=ready();Uri uri=FileProvider.getUriForFile(activity,activity.getPackageName()+".files",file);
   Intent i=new Intent(Intent.ACTION_VIEW).setDataAndType(uri,"application/vnd.android.package-archive").addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);i.setClipData(ClipData.newUri(activity.getContentResolver(),"알파 앱 업데이트",uri));
   write(current,"launched");installerLaunched=true;activity.startActivity(i);
  }catch(Exception e){installerLaunched=false;say("설치 화면을 열지 못했습니다. 앱 메뉴의 업데이트 확인으로 다시 시도해 주세요.");}
 }
}

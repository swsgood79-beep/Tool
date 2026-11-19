package com.alpharobotics.admin;
import android.Manifest;
import android.app.*;
import android.content.*;
import android.content.pm.PackageManager;
import android.os.Build;
import android.net.Uri;
import com.google.firebase.messaging.FirebaseMessagingService;
import com.google.firebase.messaging.RemoteMessage;
import java.util.HashMap;
import java.util.Map;
public final class AdminMessagingService extends FirebaseMessagingService {
 public static final String PURCHASE_CHANNEL="purchasing_calls_v1";
 public static void purchaseChannel(Context c){NotificationChannel ch=new NotificationChannel(PURCHASE_CHANNEL,"구매팀 호출",NotificationManager.IMPORTANCE_HIGH);ch.setDescription("구매 담당자에게 접수된 호출 알림");ch.enableVibration(true);ch.setLockscreenVisibility(Notification.VISIBILITY_PRIVATE);((NotificationManager)c.getSystemService(NOTIFICATION_SERVICE)).createNotificationChannel(ch);}
 public static final String CHANNEL="inspection_requests_v1";
 public static void channel(Context c){NotificationChannel ch=new NotificationChannel(CHANNEL,"검사 신청·비고 알림",NotificationManager.IMPORTANCE_HIGH);ch.setDescription("알파 관리자 전용 알림");ch.enableVibration(true);ch.setLockscreenVisibility(Notification.VISIBILITY_PRIVATE);((NotificationManager)c.getSystemService(NOTIFICATION_SERVICE)).createNotificationChannel(ch);}
 @Override public void onNewToken(String value){Store s=new Store(this);s.put("fcm_token",value);s.put("registered_token","");if(s.flag("user_disabled")||s.flag("signed_out")||s.secret().isEmpty())return;new Thread(()->{try{String base=Net.endpoint();Map<String,String> f=new HashMap<>();f.put("device",s.device());f.put("secret",s.secret());f.put("token",value);Net.Response r=Net.request("POST",base+"/native-admin/token-refresh",null,f);if(r.status==200){s.put("endpoint",base);s.put("registered_token",value);}}catch(Exception ignored){}},"alpha-token-refresh").start();}
 @Override public void onMessageReceived(RemoteMessage message){
  Store s=new Store(this);Map<String,String>d=message.getData();
  if(!RecipientPolicy.accepts(s.flag("signed_out"),s.flag("user_disabled"),s.get("account_role"),s.get("login_account"),d.get("kind"),d.get("account_key")))return;
  if(Build.VERSION.SDK_INT>=33&&checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS)!=PackageManager.PERMISSION_GRANTED)return;
  boolean purchase="purchasing_call".equals(d.get("kind"));String channel=purchase?PURCHASE_CHANNEL:CHANNEL;
  if(purchase)purchaseChannel(this);else channel(this);
  long project=number(d.get("project_id")),request=number(d.get("request_id")),call=number(d.get("call_id"));
  Intent tap=new Intent(this,MainActivity.class).putExtra("project_id",project).putExtra("request_id",request).putExtra("call_id",call)
   .setData(Uri.parse(purchase?"alpha-admin://purchase/"+call:"alpha-admin://inspection/"+project+"/"+request)).addFlags(Intent.FLAG_ACTIVITY_CLEAR_TOP|Intent.FLAG_ACTIVITY_SINGLE_TOP);
  String tag=d.getOrDefault("tag",purchase?"alpha-purchasing":"alpha-admin");int id=tag.hashCode()&0x7fffffff;
  PendingIntent pi=PendingIntent.getActivity(this,id,tap,PendingIntent.FLAG_UPDATE_CURRENT|PendingIntent.FLAG_IMMUTABLE);
  String label=purchase?"알파 구매팀":"알파 관리자";
  Notification publicNotice=new Notification.Builder(this,channel).setSmallIcon(R.drawable.ic_notification).setContentTitle(label).setContentText(purchase?"새 구매팀 호출이 있습니다.":"새 관리자 알림이 있습니다.").build();
  Notification n=new Notification.Builder(this,channel).setSmallIcon(R.drawable.ic_notification).setContentTitle(d.getOrDefault("title",label)).setContentText(d.getOrDefault("body",purchase?"새 호출이 있습니다.":"새 검사 신청이 있습니다."))
   .setStyle(new Notification.BigTextStyle().bigText(d.getOrDefault("body",""))).setAutoCancel(true).setContentIntent(pi).setVisibility(Notification.VISIBILITY_PRIVATE).setPublicVersion(publicNotice).setCategory(Notification.CATEGORY_MESSAGE).build();
  ((NotificationManager)getSystemService(NOTIFICATION_SERVICE)).notify(tag,id,n);
 }

 private static long number(String s){try{return Math.max(0,Long.parseLong(s));}catch(Exception e){return 0;}}
}

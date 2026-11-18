package com.alpharobotics.admin;
import android.content.Context;
import com.google.firebase.FirebaseApp;
import com.google.firebase.FirebaseOptions;
import com.google.firebase.messaging.FirebaseMessaging;
import org.json.JSONObject;
final class FirebaseSetup {
 static synchronized void initialize(Context c)throws Exception{Store s=new Store(c);if(s.get("firebase").isEmpty()||s.flag("user_disabled")||s.flag("signed_out"))return;JSONObject j=new JSONObject(s.get("firebase"));FirebaseOptions opts=new FirebaseOptions.Builder().setProjectId(j.getString("ProjectID")).setApplicationId(j.getString("ApplicationID")).setGcmSenderId(j.getString("SenderID")).setApiKey(j.getString("APIKey")).build();FirebaseApp app=null;try{app=FirebaseApp.getInstance();}catch(IllegalStateException ignored){}if(app!=null&&!app.getOptions().getApplicationId().equals(opts.getApplicationId())){app.delete();app=null;}if(app==null)FirebaseApp.initializeApp(c,opts);FirebaseMessaging.getInstance().setAutoInitEnabled(true);}
}

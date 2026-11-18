package com.alpharobotics.admin;
import android.app.Application;
public final class AlphaApplication extends Application {
 @Override public void onCreate(){super.onCreate();AdminMessagingService.channel(this);try{FirebaseSetup.initialize(this);}catch(Exception ignored){}}
}

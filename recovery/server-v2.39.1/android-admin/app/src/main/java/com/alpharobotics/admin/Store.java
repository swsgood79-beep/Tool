package com.alpharobotics.admin;
import android.content.Context;
import android.content.SharedPreferences;
import android.security.keystore.KeyGenParameterSpec;
import android.security.keystore.KeyProperties;
import android.util.Base64;
import java.security.KeyStore;
import java.security.SecureRandom;
import javax.crypto.Cipher;
import javax.crypto.KeyGenerator;
import javax.crypto.SecretKey;
import javax.crypto.spec.GCMParameterSpec;
final class Store {
 private final SharedPreferences p;
 Store(Context c){p=c.getSharedPreferences("alpha-admin",Context.MODE_PRIVATE);}
 String get(String key){return p.getString(key,"");} void put(String key,String value){p.edit().putString(key,value).apply();}
 boolean flag(String key){return p.getBoolean(key,false);}void flag(String key,boolean v){p.edit().putBoolean(key,v).apply();}
 String device(){String v=get("device");if(v.length()!=43){byte[] b=new byte[32];new SecureRandom().nextBytes(b);v=Base64.encodeToString(b,Base64.URL_SAFE|Base64.NO_WRAP|Base64.NO_PADDING);put("device",v);}return v;}
 private static synchronized SecretKey key()throws Exception{KeyStore k=KeyStore.getInstance("AndroidKeyStore");k.load(null);if(!k.containsAlias("alpha-admin-secret")){KeyGenerator g=KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES,"AndroidKeyStore");g.init(new KeyGenParameterSpec.Builder("alpha-admin-secret",KeyProperties.PURPOSE_ENCRYPT|KeyProperties.PURPOSE_DECRYPT).setBlockModes(KeyProperties.BLOCK_MODE_GCM).setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE).build());g.generateKey();}return (SecretKey)k.getKey("alpha-admin-secret",null);}
 void secret(String value){putSecret("device_secret",value);}
 void putSecret(String name,String value){if(value.isEmpty()){put(name,"");return;}try{Cipher c=Cipher.getInstance("AES/GCM/NoPadding");c.init(Cipher.ENCRYPT_MODE,key());put(name,Base64.encodeToString(c.getIV(),Base64.NO_WRAP)+":"+Base64.encodeToString(c.doFinal(value.getBytes(java.nio.charset.StandardCharsets.UTF_8)),Base64.NO_WRAP));}catch(Exception e){throw new IllegalStateException("앱 보안 저장소를 사용할 수 없습니다.");}}
 String secret(){return getSecret("device_secret");}
 String getSecret(String name){try{String[] v=get(name).split(":",2);if(v.length!=2)return "";Cipher c=Cipher.getInstance("AES/GCM/NoPadding");c.init(Cipher.DECRYPT_MODE,key(),new GCMParameterSpec(128,Base64.decode(v[0],Base64.NO_WRAP)));return new String(c.doFinal(Base64.decode(v[1],Base64.NO_WRAP)),java.nio.charset.StandardCharsets.UTF_8);}catch(Exception e){return "";}}
}

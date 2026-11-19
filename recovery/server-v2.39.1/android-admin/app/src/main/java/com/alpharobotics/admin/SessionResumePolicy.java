package com.alpharobotics.admin;
/** A saved grant permits server validation, never an unverified local unlock. */
final class SessionResumePolicy {
 static boolean saved(boolean signedOut,String grant,String secret){return !signedOut&&grant!=null&&!grant.isEmpty()&&secret!=null&&!secret.isEmpty();}
 static boolean start(boolean foreground,boolean open,boolean busy,boolean waiting,boolean signedOut,String grant,String secret){return foreground&&!open&&!busy&&waiting&&saved(signedOut,grant,secret);}
}

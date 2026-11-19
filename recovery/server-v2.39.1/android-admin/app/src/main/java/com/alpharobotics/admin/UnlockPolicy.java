package com.alpharobotics.admin;
/** Process-local authentication is reset on background entry and is never persisted. */
final class UnlockPolicy {
 private boolean open;
 boolean isOpen(){return open;}
 void lock(){open=false;}
 void authenticated(){open=true;}
 static boolean needsUnlock(boolean hasGrant,boolean enabled){return hasGrant&&enabled;}
}

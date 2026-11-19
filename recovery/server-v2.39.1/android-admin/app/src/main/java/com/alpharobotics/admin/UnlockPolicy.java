package com.alpharobotics.admin;
/** Process-local verified authentication survives background entry; logout/revocation locks it. Never persisted. */
final class UnlockPolicy {
 private boolean open;
 boolean isOpen(){return open;}
 void lock(){open=false;}
 void authenticated(){open=true;}
}

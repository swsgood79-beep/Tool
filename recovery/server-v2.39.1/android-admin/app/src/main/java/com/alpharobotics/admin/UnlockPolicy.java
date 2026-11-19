package com.alpharobotics.admin;
/** A verified screen remains open across background entry; only explicit lock/revocation closes it. */
final class UnlockPolicy {
 private boolean open;
 boolean isOpen(){return open;}
 void lock(){open=false;}
 void background(boolean choosingFile){/* Backgrounding never logs the user out. */}
 void authenticated(){open=true;}
}

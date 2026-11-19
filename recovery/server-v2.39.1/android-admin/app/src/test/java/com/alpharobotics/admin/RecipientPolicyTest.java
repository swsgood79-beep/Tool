package com.alpharobotics.admin;
import org.junit.Test;
import static org.junit.Assert.*;
public class RecipientPolicyTest {
 @Test public void purchasingReceivesOnlyOwnCalls(){assertTrue(RecipientPolicy.accepts(false,false,"purchasing","buyer","purchasing_call","buyer"));assertFalse(RecipientPolicy.accepts(false,false,"purchasing","buyer",null,"buyer"));assertFalse(RecipientPolicy.accepts(false,false,"purchasing","buyer","purchasing_call","other"));assertFalse(RecipientPolicy.accepts(false,false,"purchasing","buyer","purchasing_call",null));}
 @Test public void administratorsKeepExistingMessages(){assertTrue(RecipientPolicy.accepts(false,false,"admin","",null,""));assertTrue(RecipientPolicy.accepts(false,false,"",null,null,null));assertFalse(RecipientPolicy.accepts(false,false,"admin","","purchasing_call",""));assertFalse(RecipientPolicy.accepts(false,false,"admin","qa1",null,"qa2"));}
 @Test public void logoutAndMuteSuppressCalls(){assertFalse(RecipientPolicy.accepts(true,false,"purchasing","buyer","purchasing_call","buyer"));assertFalse(RecipientPolicy.accepts(false,true,"purchasing","buyer","purchasing_call","buyer"));}
 @Test public void tapDestinationIsBoundedNumeric(){assertEquals("/purchasing",UrlPolicy.purchasing(0));assertEquals("/purchasing",UrlPolicy.purchasing(-1));assertEquals("/purchasing?call=42",UrlPolicy.purchasing(42));}
}

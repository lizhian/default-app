#import <Cocoa/Cocoa.h>
#import <CoreServices/CoreServices.h>
#include <stdlib.h>

static char *json(id object) {
 if (!object) return NULL;
 NSData *data=[NSJSONSerialization dataWithJSONObject:object options:0 error:NULL];
 if (!data) return NULL;
 return strndup(data.bytes,data.length);
}
static NSURL *applicationURL(NSString *identifier) {
 NSArray *urls=CFBridgingRelease(LSCopyApplicationURLsForBundleIdentifier((__bridge CFStringRef)identifier,NULL));
 return urls.firstObject;
}
static NSDictionary *application(NSString *identifier) {
 if (!identifier.length) return @{@"id":@"",@"name":@"未设置",@"path":@""};
 NSURL *url=applicationURL(identifier);
 NSString *name=url ? [[NSFileManager defaultManager] displayNameAtPath:url.path] : identifier;
 return @{@"id":identifier,@"name":name?:identifier,@"path":url.path?:@""};
}
char *da_resolve(const char *extension) {
 @autoreleasepool {
  NSString *ext=[NSString stringWithUTF8String:extension];
  CFStringRef raw=UTTypeCreatePreferredIdentifierForTag(kUTTagClassFilenameExtension,(__bridge CFStringRef)ext,NULL);
  if (!raw) return NULL;
  NSString *type=CFBridgingRelease(raw);
  NSString *name=CFBridgingRelease(UTTypeCopyDescription((__bridge CFStringRef)type));
  NSArray *extensions=CFBridgingRelease(UTTypeCopyAllTagsWithClass((__bridge CFStringRef)type,kUTTagClassFilenameExtension));
  NSMutableArray *roles=[NSMutableArray array];
  LSRolesMask masks[]={kLSRolesViewer,kLSRolesEditor,kLSRolesShell,kLSRolesAll};
  for (int i=0;i<4;i++) {
   NSString *handler=CFBridgingRelease(LSCopyDefaultRoleHandlerForContentType((__bridge CFStringRef)type,masks[i]));
   [roles addObject:handler?:@""];
  }
  NSString *current=CFBridgingRelease(LSCopyDefaultRoleHandlerForContentType((__bridge CFStringRef)type,kLSRolesAll));
  return json(@{@"id":type,@"name":name?:@"自定义文件类型",@"extensions":extensions?:@[ext],@"handlers":roles,@"default":application(current)});
 }
}
char *da_applications(const char *identifier) {
 @autoreleasepool {
  NSString *type=[NSString stringWithUTF8String:identifier];
  NSArray *ids=CFBridgingRelease(LSCopyAllRoleHandlersForContentType((__bridge CFStringRef)type,kLSRolesAll));
  NSMutableArray *apps=[NSMutableArray array];
  for (NSString *item in ids) {
   NSDictionary *app=application(item);
   if ([app[@"path"] length]) [apps addObject:app];
  }
  return json(apps);
 }
}
char *da_application(const char *path) {
 @autoreleasepool {
  NSString *p=[NSString stringWithUTF8String:path];
  NSBundle *bundle=[NSBundle bundleWithPath:p];
  if (![p.pathExtension.lowercaseString isEqualToString:@"app"] || !bundle.bundleIdentifier.length || !bundle.executablePath || ![[NSFileManager defaultManager] isExecutableFileAtPath:bundle.executablePath]) return NULL;
  return json(@{@"id":bundle.bundleIdentifier,@"name":[[NSFileManager defaultManager] displayNameAtPath:p],@"path":p});
 }
}
int da_set(const char *identifier,int role,const char *app) {
 @autoreleasepool {
  if (role<0 || role>3) return paramErr;
  LSRolesMask masks[]={kLSRolesViewer,kLSRolesEditor,kLSRolesShell,kLSRolesAll};
  NSString *type=[NSString stringWithUTF8String:identifier], *bundle=[NSString stringWithUTF8String:app];
  if (!applicationURL(bundle)) return kLSApplicationNotFoundErr;
  return LSSetDefaultRoleHandlerForContentType((__bridge CFStringRef)type,masks[role],(__bridge CFStringRef)bundle);
 }
}
int da_register(const char *path) {
 @autoreleasepool {return LSRegisterURL((__bridge CFURLRef)[NSURL fileURLWithPath:[NSString stringWithUTF8String:path]],true);}
}

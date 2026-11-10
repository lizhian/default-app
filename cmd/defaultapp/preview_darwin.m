#import <Cocoa/Cocoa.h>
#import <CoreServices/CoreServices.h>
#include <string.h>

char *defaultapp_preview_candidates(const char *pathsJSON, const char *typesJSON) {
 @autoreleasepool {
  NSArray *paths=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:pathsJSON] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:nil];
  NSArray *types=[NSJSONSerialization JSONObjectWithData:[[NSString stringWithUTF8String:typesJSON] dataUsingEncoding:NSUTF8StringEncoding] options:0 error:nil];
  NSMutableDictionary *result=[NSMutableDictionary dictionary];
  for (NSString *path in paths) {
   NSBundle *bundle=[NSBundle bundleWithPath:path];
   NSDictionary *info=bundle.infoDictionary;
   NSArray *supported=info[@"NSExtension"][@"NSExtensionAttributes"][@"QLSupportedContentTypes"];
   if (![supported isKindOfClass:[NSArray class]]) continue;
   NSString *owner=path;
   while (owner.length>1 && ![owner.pathExtension isEqualToString:@"app"]) owner=owner.stringByDeletingLastPathComponent;
   if (owner.length<=1) owner=path;
   NSString *name=[[NSFileManager defaultManager] displayNameAtPath:owner];
   NSString *identifier=bundle.bundleIdentifier ?: path.lastPathComponent;
   for (NSString *type in types) {
    if (!type.length) continue;
    BOOL matches=NO;
    for (id candidate in supported) {
     if ([candidate isKindOfClass:[NSString class]] && UTTypeConformsTo((__bridge CFStringRef)type,(__bridge CFStringRef)candidate)) { matches=YES; break; }
    }
    if (!matches) continue;
    NSMutableArray *apps=result[type];
    if (!apps) { apps=[NSMutableArray array]; result[type]=apps; }
    BOOL duplicate=NO;
    for (NSDictionary *app in apps) if ([app[@"id"] isEqualToString:identifier]) duplicate=YES;
    if (!duplicate) [apps addObject:@{@"id":identifier,@"name":name ?: identifier,@"path":owner}];
   }
  }
  NSData *data=[NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
  return data ? strdup([[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String) : NULL;
 }
}

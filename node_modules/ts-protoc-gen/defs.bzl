DEPRECATION_WARNING = """
###################################################################################################

The bazel rules have been moved to a separate project:
https://github.com/Dig-Doug/rules_typescript_proto

For a migration guide, see:
https://github.com/Dig-Doug/rules_typescript_proto/blob/master/docs/migrating_from_ts_protoc_gen.md

###################################################################################################
"""

def typescript_proto_library(name, proto):
    print(DEPRECATION_WARNING)

def typescript_proto_dependencies():
    print(DEPRECATION_WARNING)

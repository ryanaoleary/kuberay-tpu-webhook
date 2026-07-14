import sys

from google.protobuf.descriptor import FieldDescriptor

# This patch is necessary to make newer vLLM nightly Docker images compatible with older Ray.
# Newer vLLM nightlies require and compile with protobuf 5.x. However, Ray Serve's configuration
# parser still relies on the deprecated `FieldDescriptor.label` attribute, triggering an
# AttributeError during Ray Serve application deployment.
if not hasattr(FieldDescriptor, "label"):
    try:
        FieldDescriptor.label = property(lambda self: self._label)
        print(
            "Successfully patched FieldDescriptor.label for protobuf 5.x compatibility",
            file=sys.stderr,
        )
    except Exception as e:
        print(f"Failed to patch FieldDescriptor.label: {e}", file=sys.stderr)

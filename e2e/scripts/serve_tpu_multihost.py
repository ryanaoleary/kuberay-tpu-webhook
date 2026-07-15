import os

from ray.serve.llm import LLMConfig, LLMServingArgs, build_openai_app

MODEL_ID = os.environ.get("MODEL_ID", "Qwen/Qwen1.5-0.5B-Chat")
ACCELERATOR_TYPE = os.environ.get("ACCELERATOR_TYPE", "TPU-V6E")
TPU_TOPOLOGY = os.environ.get("TPU_TOPOLOGY", "")
TENSOR_PARALLEL_SIZE = int(os.environ.get("TENSOR_PARALLEL_SIZE", "4"))
MAX_MODEL_LEN = int(os.environ.get("MAX_MODEL_LEN", "4096"))

accelerator_config = {"kind": "tpu"}
if TPU_TOPOLOGY:
    accelerator_config["topology"] = TPU_TOPOLOGY

llm_config = LLMConfig(
    model_loading_config=dict(
        model_id=MODEL_ID,
    ),
    accelerator_type=ACCELERATOR_TYPE,
    accelerator_config=accelerator_config,
    engine_kwargs={
        "tensor_parallel_size": TENSOR_PARALLEL_SIZE,
        "max_model_len": MAX_MODEL_LEN,
        "distributed_executor_backend": "ray",
        "enforce_eager": True,
        "load_format": "pathways_dummy",
    },
)

deployment = build_openai_app(LLMServingArgs(llm_configs=[llm_config]))

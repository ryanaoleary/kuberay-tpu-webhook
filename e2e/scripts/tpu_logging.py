import logging
import sys


def setup_tpu_logging(logger_name: str) -> logging.Logger:
    """Configures standard logging to stdout with structured formatting for E2E TPU workloads."""
    logging.basicConfig(
        level=logging.INFO,
        format="%(levelname)s: %(message)s",
        handlers=[logging.StreamHandler(sys.stdout)],
    )
    return logging.getLogger(logger_name)

# Build container for the advanced_pytorch example according to the following instructions:
# https://github.com/adap/flower/tree/main/examples/advanced_pytorch
FROM bitnami/pytorch

USER root

RUN apt-get update && apt-get install -y git

# Install dependencies
RUN pip install poetry

# Clone the example project
RUN git clone --depth=1 https://github.com/adap/flower.git  \
    && mv flower/examples/advanced_pytorch .                \
    && rm -rf flower

WORKDIR ./advanced_pytorch

# Download Flower dependencies
RUN poetry install

# Download the EfficientNetB0 model
RUN python -c "import torch; torch.hub.load( \
            'NVIDIA/DeepLearningExamples:torchhub', \
            'nvidia_efficientnet_b0', pretrained=True)"

# Download the CIFAR-10 dataset
RUN python -c "from torchvision.datasets import CIFAR10; CIFAR10('./dataset', download=True)"


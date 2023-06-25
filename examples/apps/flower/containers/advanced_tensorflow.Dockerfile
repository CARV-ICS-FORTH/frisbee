# Build container for the advanced_pytorch example according to the following instructions:
# https://github.com/adap/flower/tree/main/examples/advanced_tensorflow
FROM tensorflow/tensorflow

USER root

RUN apt-get update && apt-get install -y git

# Install dependencies
RUN pip install poetry

# Clone the example project
RUN git clone --depth=1 https://github.com/adap/flower.git      \
    && mv flower/examples/advanced_tensorflow .                 \
    && rm -rf flower

WORKDIR ./advanced_tensorflow

# Download Flower dependencies
RUN poetry install

# Download the CIFAR-10 dataset
RUN python -c "import tensorflow as tf; tf.keras.datasets.cifar10.load_data()"
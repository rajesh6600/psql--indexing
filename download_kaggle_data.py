import kagglehub

# Specify the dataset you want to download
dataset_id = "olistbr/brazilian-ecommerce"

# Download the dataset
path = kagglehub.dataset_download(dataset_id)

# Print the path to the downloaded dataset files
print("Path to dataset files:", path)

package main

import (
	"github.com/kahvecikaan/buildingMicroservices/product-api/sdk/client"
	"github.com/kahvecikaan/buildingMicroservices/product-api/sdk/client/products"
	"github.com/kahvecikaan/buildingMicroservices/product-api/sdk/models"
	"testing"
)

func TestProductAPI(t *testing.T) {
	cfg := client.DefaultTransportConfig().WithHost("localhost:9090")
	c := client.NewHTTPClientWithConfig(nil, cfg)

	t.Run("ListProducts", func(t *testing.T) {
		params := products.NewListProductsParams()
		resp, err := c.Products.ListProducts(params)
		if err != nil {
			t.Fatalf("Error listing products: %v", err)
		}
		if len(resp.Payload) == 0 {
			t.Fatalf("No products returned")
		}
	})

	t.Run("CreateUpdateDeleteProduct", func(t *testing.T) {
		// Create a product
		name := "Test Product"
		price := float32(9.99)
		sku := "test-sku-ddd"
		newProduct := &models.Product{
			Name:  &name,
			Price: &price,
			SKU:   &sku,
		}
		createParams := products.NewCreateProductParams().WithBody(newProduct)
		createResp, err := c.Products.CreateProduct(createParams)
		if err != nil {
			t.Fatalf("Error creating product: %v", err)
		}
		if createResp.Payload.ID == nil || *createResp.Payload.ID == 0 {
			t.Fatalf("Created product ID is nil or 0")
		}

		// Update the product
		updatedName := "Updated Test Product"
		updatedPrice := float32(19.99)
		updatedProduct := &models.Product{
			ID:    createResp.Payload.ID,
			Name:  &updatedName,
			Price: &updatedPrice,
			SKU:   &sku,
		}
		updateParams := products.NewUpdateProductParams().WithBody(updatedProduct)
		_, err = c.Products.UpdateProduct(updateParams)
		if err != nil {
			// Check if it's a 204 No Content response
			if _, ok := err.(*products.UpdateProductNoContent); !ok {
				t.Fatalf("Error updating product: %v", err)
			}
		}

		// Get the updated product
		getParams := products.NewListSingleProductParams().WithID(*createResp.Payload.ID)
		getResp, err := c.Products.ListSingleProduct(getParams)
		if err != nil {
			t.Fatalf("Error getting updated product: %v", err)
		}
		if *getResp.Payload.Name != updatedName || *getResp.Payload.Price != updatedPrice {
			t.Fatalf("Product was not updated correctly")
		}

		// Delete the product
		deleteParams := products.NewDeleteProductParams().WithID(*createResp.Payload.ID)
		_, err = c.Products.DeleteProduct(deleteParams)
		if err != nil {
			// Check if it's a 204 No Content response
			if _, ok := err.(*products.DeleteProductNoContent); !ok {
				t.Fatalf("Error deleting test product: %v", err)
			}
		}

		// Verify the product is deleted
		_, err = c.Products.ListSingleProduct(getParams)
		if err == nil {
			t.Fatalf("Product was not deleted")
		}
	})
}

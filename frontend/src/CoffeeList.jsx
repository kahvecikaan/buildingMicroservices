import React, { useState, useEffect } from 'react';
import { Table, Card } from 'react-bootstrap';
import axios from 'axios';

function CoffeeList() {
    const [products, setProducts] = useState([]);

    useEffect(() => {
        axios.get(`${process.env.REACT_APP_API_LOCATION}/products`)
            .then(response => {
                console.log(response.data);
                setProducts(response.data);
            })
            .catch(error => {
                console.log(error);
            });
    }, []);

    return (
        <Card>
            <Card.Header as="h5" className="bg-primary text-white">Our Coffee Menu</Card.Header>
            <Card.Body>
                <Table striped bordered hover>
                    <thead>
                    <tr>
                        <th>Name</th>
                        <th>Price</th>
                        <th>SKU</th>
                    </tr>
                    </thead>
                    <tbody>
                    {products.map((product, index) => (
                        <tr key={index}>
                            <td>{product.name}</td>
                            <td>£{product.price.toFixed(2)}</td>
                            <td>{product.sku}</td>
                        </tr>
                    ))}
                    </tbody>
                </Table>
            </Card.Body>
        </Card>
    );
}

export default CoffeeList;
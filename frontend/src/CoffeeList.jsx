import React, { useState, useEffect, useRef } from 'react';
import { Table, Card, Form, Alert, Spinner } from 'react-bootstrap';
import axios from 'axios';
import { w3cwebsocket as W3CWebSocket } from 'websocket';

function CoffeeList() {
    // State Variables
    const [products, setProducts] = useState([]);
    const [currencies, setCurrencies] = useState([]);
    const [selectedCurrency, setSelectedCurrency] = useState('');
    const [error, setError] = useState(null);
    const [loadingCurrencies, setLoadingCurrencies] = useState(true);
    const [loadingProducts, setLoadingProducts] = useState(false);

    // WebSocket Reference
    const wsClient = useRef(null);

    // Ref to track the latest selected currency
    const selectedCurrencyRef = useRef(selectedCurrency);

    // Update the ref whenever selectedCurrency changes
    useEffect(() => {
        selectedCurrencyRef.current = selectedCurrency;
    }, [selectedCurrency]);

    // Fetch available currencies when the component mounts
    useEffect(() => {
        const fetchCurrencies = async () => {
            try {
                setLoadingCurrencies(true);
                const response = await axios.get(`${process.env.REACT_APP_API_LOCATION}/currencies`);
                setCurrencies(response.data);
                if (response.data.length > 0) {
                    // Prioritize EUR as the default currency
                    const eurExists = response.data.includes('EUR');
                    setSelectedCurrency(eurExists ? 'EUR' : response.data[0]);
                }
                setError(null);
            } catch (err) {
                console.error('Error fetching currencies:', err);
                setError('Failed to load currencies. Please try again later.');
            } finally {
                setLoadingCurrencies(false);
            }
        };

        fetchCurrencies();
    }, []);

    // Function to fetch products based on the selected currency
    const fetchProducts = async (currency) => {
        try {
            setLoadingProducts(true);
            const url = currency
                ? `${process.env.REACT_APP_API_LOCATION}/products?currency=${currency}`
                : `${process.env.REACT_APP_API_LOCATION}/products`;

            const response = await axios.get(url);
            setProducts(response.data);
            setError(null);
        } catch (err) {
            console.error('Error fetching products:', err);
            setError('Failed to load products. Please try again later.');
        } finally {
            setLoadingProducts(false);
        }
    };

    // Fetch products whenever the selected currency changes
    useEffect(() => {
        if (selectedCurrency) {
            fetchProducts(selectedCurrency);
        }
    }, [selectedCurrency]);

    // Set up WebSocket connection once when the component mounts
    useEffect(() => {
        const wsLocation = process.env.REACT_APP_WS_LOCATION;
        wsClient.current = new W3CWebSocket(wsLocation);

        wsClient.current.onopen = () => {
            console.log('WebSocket Client Connected');
        };

        wsClient.current.onmessage = (message) => {
            try {
                const parsedMessage = JSON.parse(message.data);
                if (parsedMessage['event-type'] === 'price_update') {
                    const { product_id, new_price, currency } = parsedMessage.data;

                    // Access the latest selected currency using the ref
                    const currentCurrency = selectedCurrencyRef.current;

                    // Update product prices if the currency matches the selected currency
                    if (currency === currentCurrency) {
                        setProducts(prevProducts => prevProducts.map(product => (
                            product.id === product_id
                                ? { ...product, price: new_price }
                                : product
                        )));
                    }
                }
            } catch (err) {
                console.error('Error parsing WebSocket message:', err);
            }
        };

        wsClient.current.onerror = (err) => {
            console.error('WebSocket Error:', err);
            setError('WebSocket connection error. Real-time updates may not work.');
        };

        wsClient.current.onclose = () => {
            console.log('WebSocket Client Disconnected');
        };

        // Clean up the WebSocket connection when the component unmounts
        return () => {
            if (wsClient.current) {
                wsClient.current.close();
            }
        };
    }, []); // Empty dependency array ensures this runs once on mount

    // Handle the currency selection change
    const handleCurrencyChange = (e) => {
        setSelectedCurrency(e.target.value);
    };

    return (
        <Card>
            <Card.Header as="h5" className="bg-primary text-white">
                Our Coffee Menu
            </Card.Header>
            <Card.Body>
                {/* Display Error Alert */}
                {error && (
                    <Alert variant="danger">
                        {error}
                    </Alert>
                )}

                {/* Currency Selector */}
                <Form.Group controlId="currencySelect">
                    <Form.Label>Select Currency</Form.Label>
                    {loadingCurrencies ? (
                        <div>
                            <Spinner animation="border" size="sm" /> Loading currencies...
                        </div>
                    ) : (
                        <Form.Control as="select" value={selectedCurrency} onChange={handleCurrencyChange}>
                            {currencies.map((currency) => (
                                <option key={currency} value={currency}>{currency}</option>
                            ))}
                        </Form.Control>
                    )}
                </Form.Group>

                {/* Products Table */}
                <div className="mt-3">
                    {loadingProducts ? (
                        <div>
                            <Spinner animation="border" /> Loading products...
                        </div>
                    ) : (
                        <Table striped bordered hover>
                            <thead>
                            <tr>
                                <th>Name</th>
                                <th>Price ({selectedCurrency || 'EUR'})</th>
                                <th>SKU</th>
                            </tr>
                            </thead>
                            <tbody>
                            {products.map((product) => (
                                <tr key={product.id}>
                                    <td>{product.name}</td>
                                    <td>{product.price ? product.price.toFixed(2) : 'N/A'}</td>
                                    <td>{product.sku}</td>
                                </tr>
                            ))}
                            </tbody>
                        </Table>
                    )}
                </div>
            </Card.Body>
        </Card>
    );
}

export default CoffeeList;

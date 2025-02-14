package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
)

// TestCase represents the JSON test case structure
type TestCase struct {
	Keys struct {
		N int `json:"n"` // Total number of shares
		K int `json:"k"` // Threshold
	} `json:"keys"`
	Data map[string]struct {
		Base  string `json:"base"`
		Value string `json:"value"`
	} `json:"-"`
}

// Parses JSON test cases
func parseJSON(filename string) (TestCase, error) {
	var testCase TestCase
	file, err := os.ReadFile(filename)
	if err != nil {
		return testCase, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(file, &raw); err != nil {
		return testCase, err
	}

	// Extract keys
	if keys, ok := raw["keys"].(map[string]interface{}); ok {
		testCase.Keys.N = int(keys["n"].(float64))
		testCase.Keys.K = int(keys["k"].(float64))
	}

	// Extract data points
	testCase.Data = make(map[string]struct {
		Base  string `json:"base"`
		Value string `json:"value"`
	})

	for key, val := range raw {
		if key == "keys" {
			continue
		}
		obj := val.(map[string]interface{})
		testCase.Data[key] = struct {
			Base  string `json:"base"`
			Value string `json:"value"`
		}{
			Base:  obj["base"].(string),
			Value: obj["value"].(string),
		}
	}

	return testCase, nil
}

// Decodes a string value to big.Int based on its base
func decodeBase(value string, base int) *big.Int {
	bigInt := new(big.Int)
	bigInt.SetString(value, base)
	return bigInt
}

// Performs Lagrange Interpolation in a finite field (mod prime)
func lagrangeInterpolation(points []struct{ x, y *big.Int }, prime *big.Int) *big.Int {
	secret := new(big.Int)

	for i := 0; i < len(points); i++ {
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for j := 0; j < len(points); j++ {
			if i != j {
				num := new(big.Int).Set(points[j].x)
				denom := new(big.Int).Sub(points[i].x, points[j].x)

				// Ensure denominator is positive within modular field
				denom.Mod(denom, prime)
				if denom.Sign() == 0 {
					fmt.Println("Error: Division by zero in Lagrange interpolation")
					return nil
				}

				denominator.Mul(denominator, denom)
				denominator.Mod(denominator, prime)

				numerator.Mul(numerator, num)
				numerator.Mod(numerator, prime)
			}
		}

		// Compute modular inverse of denominator
		denominatorInv := new(big.Int).ModInverse(denominator, prime)
		if denominatorInv == nil {
			fmt.Println("Error: Modular inverse does not exist")
			return nil
		}

		term := new(big.Int).Mul(points[i].y, numerator)
		term.Mul(term, denominatorInv)
		term.Mod(term, prime)

		secret.Add(secret, term)
		secret.Mod(secret, prime)
	}

	return secret
}

func main() {
	// Define the prime modulus (2^521 - 1)
	prime := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(521), nil), big.NewInt(1))

	// List of test case files
	filenames := []string{"testcase1.json", "testcase2.json"}
	for _, filename := range filenames {
		testCase, err := parseJSON(filename)
		if err != nil {
			fmt.Println("Error reading file:", err)
			continue
		}

		// Ensure there are enough shares to reconstruct the secret
		if testCase.Keys.N < testCase.Keys.K {
			fmt.Println("Invalid test case:", filename, "- Insufficient shares to reconstruct the secret")
			continue
		}

		// Convert JSON data into (x, y) points
		var points []struct{ x, y *big.Int }
		for key, data := range testCase.Data {
			x := new(big.Int)
			x.SetString(key, 10) // Convert string key to big.Int
			base, _ := strconv.Atoi(data.Base)
			y := decodeBase(data.Value, base)
			points = append(points, struct{ x, y *big.Int }{x, y})
		}

		// Sort points based on x values
		sort.Slice(points, func(i, j int) bool {
			return points[i].x.Cmp(points[j].x) < 0
		})

		// Use only the first k shares for reconstruction
		points = points[:testCase.Keys.K]
		secret := lagrangeInterpolation(points, prime)

		fmt.Println("Reconstructed secret for", filename, "is", secret)
	}
}
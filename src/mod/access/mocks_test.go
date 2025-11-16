package access

import (
	"github.com/stretchr/testify/mock"
	"imuslab.com/zoraxy/mod/geodb"
)

// MockGeoDB is a mock implementation of geodb.Store
type MockGeoDB struct {
	mock.Mock
}

func (m *MockGeoDB) ResolveCountryCodeFromIP(ipAddr string) (*geodb.CountryInfo, error) {
	args := m.Called(ipAddr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*geodb.CountryInfo), args.Error(1)
}

func (m *MockGeoDB) GetCountryFlag(countryCode string) (string, error) {
	args := m.Called(countryCode)
	return args.String(0), args.Error(1)
}

func (m *MockGeoDB) Close() {
	m.Called()
}

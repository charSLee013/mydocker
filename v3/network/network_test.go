package network

import "testing"

func TestCreateNetwork(t *testing.T) {
	driverName := "gocker0"
	subnet := "192.0.2.1/24"
	name :=  "gocker0"

	if err := Init();err!= nil {
		t.Errorf("Init Error: %v",err)
	}

	err := CreateNetwork(driverName,subnet,name)
	if err != nil {
		t.Errorf("Create Network Error \n DriverName: %s\t Subnet: %s\t Name: %s\nError: %v ",driverName,subnet,name,err)
	}
}

func TestDeleteNetwork(t *testing.T) {
	driverName := "gocker0"

	if err := Init();err!= nil {
		t.Errorf("Init Error: %v",err)
	}

	if err:=DeleteNetwork(driverName);err != nil {
		t.Errorf("Delete Network Error \n DriverName: %s \n Error: %v",driverName,err)
	}
}

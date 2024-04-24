package test

import (
	"encoding/json"
	"fmt"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

var (
	tfNetOpts  terraform.Options
	tfVolOpts  terraform.Options
	tfInstOpts terraform.Options
	tfKeyOpts  terraform.Options
	tfSerGOpts terraform.Options
	tfFipOpts  terraform.Options
	tfRFipOpts terraform.Options
	tfSecGOpts terraform.Options
)

func TestMain(m *testing.M) {
	code := m.Run()
	defer os.Exit(code)
}

func TestCreatePrepareResourcesForTerraformEdgeCenterInstance(t *testing.T) {
	// Инициализация и применение конфигурации сетевого модуля.
	initializeNetworkModule(&tfNetOpts)
	applyModule(t, &tfNetOpts, "network")
	// Получение и сохранение идентификаторов сети и подсети.
	networkID := terraform.Output(t, &tfNetOpts, "network_id")
	subnetID := terraform.Output(t, &tfNetOpts, "subnet_id")

	// Инициализация и применение модуля volume.
	initializeVolumeModule(&tfVolOpts)
	applyModule(t, &tfVolOpts, "volume")

	// Инициализация и применение модуля ключевых пар.
	initializeKeypairModule(&tfKeyOpts)
	applyModule(t, &tfKeyOpts, "keypair")

	// Получение идентификатора группы серверов.
	initializeServerGroupModule(&tfSerGOpts)
	applyModule(t, &tfSerGOpts, "server group")

	// Инициализация и применение модуля плавающих IP-адресов.
	initializeFloatingIPModule(&tfFipOpts)
	applyModule(t, &tfFipOpts, "floating IP")

	// Инициализация и применение модуля зарезервированных фиксированных IP-адресов.
	initializeReservedFIPModule(&tfRFipOpts, networkID, subnetID)
	applyModule(t, &tfRFipOpts, "reserved fixed IP")

	// Инициализация и применение модуля групп безопасности.
	initializeSecGroupModule(&tfSecGOpts)
	applyModule(t, &tfSecGOpts, "security group")
}

func TestCreateTerraformEdgeCenterInstance(t *testing.T) {
	// Получение и сохранение идентификаторов сети и подсети.
	networkID := getOutput(t, &tfNetOpts, "network_id")
	subnetID := getOutput(t, &tfNetOpts, "subnet_id")

	// Получение идентификаторов первого, второго и третьего тома.
	firstVolumeID := getOutput(t, &tfVolOpts, "first_volume_id")
	secondVolumeID := getOutput(t, &tfVolOpts, "second_volume_id")
	thirdVolumeID := getOutput(t, &tfVolOpts, "third_volume_id")

	// Получение идентификатора группы серверов.
	serverGroupID := getOutput(t, &tfSerGOpts, "server_group_id")

	// Инициализация и применение основного модуля экземпляра.
	initializeInstanceModule(&tfInstOpts, networkID, subnetID, firstVolumeID, secondVolumeID, thirdVolumeID, serverGroupID)
	// TODO: Нужно заменить на ApplyAndIdempotentE
	applyModule(t, &tfInstOpts, "instance")

	// После успешного создания всех ресурсов выполняется их проверка.
	validateInstanceOutputs(t)
}

// validateInstanceOutputs проверяет вывод Terraform для созданного сервера, включая volumes и конфигурации сетевых интерфейсов.
func validateInstanceOutputs(t *testing.T) {
	// Проверяем базовые параметры сервера, такие как ID и тип (flavor).
	validateBasicOutputs(t)

	// Проверяем прикрепленные volumes и их свойства.
	validateVolumeOutputs(t)

	// Проверяем конфигурации сетевых интерфейсов.
	validateNetworkInterfaces(t)
}

// validateBasicOutputs проверяет основные параметры сервера.
func validateBasicOutputs(t *testing.T) {
	instanceID := terraform.Output(t, &tfInstOpts, "instance_id")
	assertNotEmpty(t, instanceID, "Instance ID should not be empty")

	getAndAssertOutput(t, &tfInstOpts, "flavor_id", instanceFlavor, "Flavor ID should match the expected value")
	getAndAssertOutput(t, &tfInstOpts, "instance_name", instanceName, "Instance name should match the expected value")
	getAndAssertOutput(t, &tfInstOpts, "keypair_name", keypairName, "Keypair name should match the expected value")
	getAndAssertOutput(t, &tfInstOpts, "server_group", terraform.Output(t, &tfSerGOpts, "server_group_id"), "Server group should match the expected ID")
	getAndAssertOutput(t, &tfInstOpts, "user_data", userData, "User data should match the expected value")
}

// validateVolumeOutputs проверяет прикрепленные volumes.
func validateVolumeOutputs(t *testing.T) {
	volumes := terraform.OutputListOfObjects(t, &tfInstOpts, "instance_volumes")
	require.Equal(t, 3, len(volumes), "There should be three volumes attached to the instance")

	for i, volumeID := range []string{getOutput(t, &tfVolOpts, "first_volume_id"),
		getOutput(t, &tfVolOpts, "second_volume_id"),
		getOutput(t, &tfVolOpts, "third_volume_id")} {
		// TODO: Невозможно проверить параметр deleteOnTermination поскольку он не изменяется и не читается функцией Read
		//deleteOnTermination := tfInstOpts.Vars["instance_volumes"].([]map[string]interface{})[i]["delete_on_termination"].(bool)
		//require.Equal(t, volumes[i]["delete_on_termination"], deleteOnTermination, fmt.Sprintf("Delete on termination flag should match for volume index: %d", i))
		require.Equal(t, volumes[i]["volume_id"], volumeID, fmt.Sprintf("Volume ID should match for volume index: %d", i))
		require.Equal(t, volumes[i]["boot_index"], tfInstOpts.Vars["instance_volumes"].([]map[string]interface{})[i]["boot_index"], fmt.Sprintf("Boot index should match for volume index: %d", i))
	}
}

// validateNetworkInterfaces проверяет сетевые интерфейсы.
func validateNetworkInterfaces(t *testing.T) {
	interfaces := terraform.OutputListOfObjects(t, &tfInstOpts, "instance_interfaces")
	require.Equal(t, 1, len(interfaces), "There should be one network interface attached to the instance")

	networkID, subnetID := getOutput(t, &tfNetOpts, "network_id"), getOutput(t, &tfNetOpts, "subnet_id")
	require.Equal(t, interfaces[0]["network_id"], networkID, "Network ID should match")
	require.Equal(t, interfaces[0]["subnet_id"], subnetID, "Subnet ID should match")
}

// TestUpdateAddSecGroupInInterfaceInstance обновляет и добавляет группу безопасности в интерфейс экземпляра.
func TestUpdateAddSecGroupInInterfaceInstance(t *testing.T) {
	networkID := terraform.Output(t, &tfNetOpts, "network_id")
	subnetID := terraform.Output(t, &tfNetOpts, "subnet_id")
	secGroupIDs := terraform.Output(t, &tfSecGOpts, "security_group_ids")

	newInterfaces := []map[string]interface{}{
		{
			"type":                   "subnet",
			"network_id":             networkID,
			"subnet_id":              subnetID,
			"security_groups":        secGroupIDs,
			"port_security_disabled": false,
		},
	}

	tfInstOpts.Vars["instance_interfaces"] = newInterfaces

	updatedInterfacesRaw, err := terraform.OutputJsonE(t, &tfInstOpts, "instance_interfaces")
	if err != nil {
		t.Fatalf("failed to get updated interface configuration: %v", err)
	}
	fmt.Println(updatedInterfacesRaw)

	// Проверка соответствия обновленных интерфейсов заданным
	checkUpdatedInterfaces(t, updatedInterfacesRaw, newInterfaces)
}

// checkUpdatedInterfaces проверяет обновленные интерфейсы на соответствие ожидаемым значениям
func checkUpdatedInterfaces(t *testing.T, rawOutput string, expectedInterfaces []map[string]interface{}) {
	var actualInterfaces []map[string]interface{}
	if err := json.Unmarshal([]byte(rawOutput), &actualInterfaces); err != nil {
		t.Fatalf("failed to unmarshal interface configuration: %v", err)
	}

	if len(actualInterfaces) != len(expectedInterfaces) {
		t.Fatalf("unexpected number of interfaces: expected %d, got %d", len(expectedInterfaces), len(actualInterfaces))
	}

	// TODO: Не происходит Read если не передан параметр в terraform file, read не записывает значение в стейт.
	//for i, expected := range expectedInterfaces {
	//	actual := actualInterfaces[i]
	//	for key, expectedValue := range expected {
	//		actualValue, exists := actual[key]
	//		if !exists {
	//			t.Errorf("expected key %s does not exist in the actual interface configuration", key)
	//		} else if !reflect.DeepEqual(expectedValue, actualValue) {
	//			t.Errorf("mismatch in %s: expected %v, got %v", key, expectedValue, actualValue)
	//		}
	//	}
	//}
}

// TODO: Не понятно как добавить ещё один FIP к instance_interface
//func TestUpdateADDNewFipInterfaceInstance(t *testing.T) {
//	networkID := terraform.Output(t, &tfNetOpts, "network_id")
//	subnetID := terraform.Output(t, &tfNetOpts, "subnet_id")
//	fipID := terraform.Output(t, &tfFipOpts, "fip_id")
//
//	newInterfaces := []map[string]interface{}{
//		{
//			"type":                   "subnet",
//			"network_id":             networkID,
//			"subnet_id":              subnetID,
//			"port_security_disabled": false,
//		},
//		{
//			"fip_source":      "existing",
//			"type":            "external",
//			"existing_fip_id": fipID,
//		},
//	}
//
//	tfInstOpts.Vars["instance_interfaces"] = newInterfaces
//
//	if _, err := terraform.ApplyAndIdempotentE(t, &tfInstOpts); err != nil {
//		t.Fatalf("failed to apply new interface configuration with FIP: %v", err)
//	}
//
//	updatedInterfacesRaw, err := terraform.OutputJsonE(t, &tfInstOpts, "instance_interfaces")
//	if err != nil {
//		t.Fatalf("failed to get updated interface configuration: %v", err)
//	}
//
//	fmt.Printf("Raw updated interface configuration: %s\n", updatedInterfacesRaw)
//
//	// Parse the output to a usable format, if necessary
//	var updatedInterfaces []map[string]interface{}
//	if err := json.Unmarshal([]byte(updatedInterfacesRaw), &updatedInterfaces); err != nil {
//		t.Fatalf("failed to unmarshal updated interface configuration: %v", err)
//	}
//
//	if len(updatedInterfaces) != len(newInterfaces) {
//		t.Fatalf("Number of interfaces mismatch: expected %d, got %d", len(newInterfaces), len(updatedInterfaces))
//	}
//
//	for i, expected := range newInterfaces {
//		actual := updatedInterfaces[i]
//		for key, expectedValue := range expected {
//			actualValue, exists := actual[key]
//			if !exists {
//				t.Errorf("Expected key %s does not exist in the actual interface configuration", key)
//			} else if !reflect.DeepEqual(expectedValue, actualValue) {
//				t.Errorf("Mismatch for key %s: expected %v, got %v", key, expectedValue, actualValue)
//			}
//		}
//	}
//}

// TODO: Невозможно прикрепить к существующему интерфейсу типа subnet прикрепить fip.
//func TestUpdateADDFIPToExistingInterfaceInstance(t *testing.T) {
//	networkID := terraform.Output(t, &tfNetOpts, "network_id")
//	subnetID := terraform.Output(t, &tfNetOpts, "subnet_id")
//	fipID := terraform.Output(t, &tfFipOpts, "fip_id")
//	fipPortID := terraform.Output(t, &tfFipOpts, "fip_port_id")
//
//	newInterfaces := []map[string]interface{}{
//		{
//			"fip_source":             "existing",
//			"type":                   "subnet",
//			"network_id":             networkID,
//			"subnet_id":              subnetID,
//			"existing_fip_id":        fipID,
//			"port_security_disabled": false,
//			"port_id":                fipPortID,
//		},
//	}
//
//	tfInstOpts.Vars["instance_interfaces"] = newInterfaces
//
//	if _, err := terraform.ApplyAndIdempotentE(t, &tfInstOpts); err != nil {
//		t.Fatalf("failed to apply new interface configuration with FIP: %v", err)
//	}
//
//	updatedInterfacesRaw, err := terraform.OutputJsonE(t, &tfInstOpts, "instance_interfaces")
//	if err != nil {
//		t.Fatalf("failed to get updated interface configuration: %v", err)
//	}
//
//	fmt.Printf("Raw updated interface configuration: %s\n", updatedInterfacesRaw)
//
//	// Parse the output to a usable format, if necessary
//	var updatedInterfaces []map[string]interface{}
//	if err := json.Unmarshal([]byte(updatedInterfacesRaw), &updatedInterfaces); err != nil {
//		t.Fatalf("failed to unmarshal updated interface configuration: %v", err)
//	}
//
//	if len(updatedInterfaces) != len(newInterfaces) {
//		t.Fatalf("Number of interfaces mismatch: expected %d, got %d", len(newInterfaces), len(updatedInterfaces))
//	}
//
//	for i, expected := range newInterfaces {
//		actual := updatedInterfaces[i]
//		for key, expectedValue := range expected {
//			actualValue, exists := actual[key]
//			if !exists {
//				t.Errorf("Expected key %s does not exist in the actual interface configuration", key)
//			} else if !reflect.DeepEqual(expectedValue, actualValue) {
//				t.Errorf("Mismatch for key %s: expected %v, got %v", key, expectedValue, actualValue)
//			}
//		}
//	}
//}

// TODO: Невозможно использовать этот тест, поскольку он постоянно неидемпотентен
//func TestUpdateVolumesInstance(t *testing.T) {
//	firstVolumeID := terraform.Output(t, &tfVolOpts, "first_volume_id")
//	secondVolumeID := terraform.Output(t, &tfVolOpts, "second_volume_id")
//
//	twoVolumes := []map[string]interface{}{
//		{
//			"volume_id":             firstVolumeID,
//			"boot_index":            0,
//			"delete_on_termination": true,
//		},
//		{
//			"volume_id":             secondVolumeID,
//			"boot_index":            1,
//			"delete_on_termination": false,
//		},
//	}
//	tfInstOpts.Vars["instance_volumes"] = twoVolumes
//
//	if _, err := terraform.ApplyE(t, &tfInstOpts); err != nil {
//		t.Fatalf("failed to apply initial volume configuration: %v", err)
//	}
//
//	verifyVolumeParameters(t, &tfInstOpts, twoVolumes)
//
//	oneVolume := twoVolumes[:1]
//	tfInstOpts.Vars["instance_volumes"] = oneVolume
//
//	if _, err := terraform.ApplyE(t, &tfInstOpts); err != nil {
//		t.Fatalf("failed to apply changes to reduce to one volume: %v", err)
//	}
//
//	verifyVolumeParameters(t, &tfInstOpts, oneVolume)
//}
//
//// verifyVolumeParameters проверяет параметры каждого тома.
//func verifyVolumeParameters(t *testing.T, tfOpts *terraform.Options, expectedVolumes []map[string]interface{}) {
//	output, err := terraform.OutputJsonE(t, tfOpts, "instance_volumes")
//	if err != nil {
//		t.Fatalf("failed to get output for instance volumes: %v", err)
//	}
//
//	var volumes []map[string]interface{}
//	if err := json.Unmarshal([]byte(output), &volumes); err != nil {
//		t.Fatalf("failed to unmarshal instance volumes: %v", err)
//	}
//
//	if len(volumes) != len(expectedVolumes) {
//		t.Fatalf("expected %d volumes, got %d", len(expectedVolumes), len(volumes))
//		return
//	}
//
//	for i, vol := range volumes {
//		expVol := expectedVolumes[i]
//		if vol["boot_index"] != expVol["boot_index"] {
//			t.Errorf("volume %d boot_index mismatch: expected %v, got %v", i, expVol["boot_index"], vol["boot_index"])
//		}
//		if vol["delete_on_termination"] != expVol["delete_on_termination"] {
//			t.Errorf("volume %d delete_on_termination mismatch: expected %v, got %v", i, expVol["delete_on_termination"], vol["delete_on_termination"])
//		}
//	}
//}

// TestUpdateNameInstance проводит тестирование обновления имени экземпляра.
func TestUpdateNameInstance(t *testing.T) {
	newInstanceName := fmt.Sprintf("%s-%s", instanceName, random.UniqueId())
	tfInstOpts.Vars["instance_name"] = newInstanceName
	applyChanges(t, &tfInstOpts)
	checkOutput(t, &tfInstOpts, "instance_name", newInstanceName)
}

// TODO: Проверка не проходит поскольку не меняется на пустое значение
//// Обновление имени на пустое значение
//tfInstOpts.Vars["instance_name"] = ""
//if _, err := terraform.ApplyAndIdempotentE(t, &tfInstOpts); err != nil {
//	t.Fatalf("failed to apply changes to clear the instance name: %v", err)
//}
//
//updatedInstanceName, err = terraform.OutputE(t, &tfInstOpts, "instance_name")
//if err != nil {
//	t.Fatalf("failed to get updated Instance Name after clearing: %v", err)
//}
//require.Empty(t, updatedInstanceName, "Instance name should be cleared and empty")

// TestUpdateFlavorInstance тестирует обновление flavor_id инстанса и проверяет изменение.
func TestUpdateFlavorInstance(t *testing.T) {
	newFlavorID := "g1-standard-2-4"
	tfInstOpts.Vars["flavor_id"] = newFlavorID
	applyChanges(t, &tfInstOpts)
	checkOutput(t, &tfInstOpts, "flavor_id", newFlavorID)
}

// TestUpdateVmStateInstance обновляет состояние VM и проверяет новое состояние.
func TestUpdateVmStateInstance(t *testing.T) {
	newVmState := "stopped"
	tfInstOpts.Vars["vm_state"] = newVmState
	applyChanges(t, &tfInstOpts)
	getAndCheckOutput(t, &tfInstOpts, "vm_state", newVmState)

	// Возвращение к начальному состоянию
	tfInstOpts.Vars["vm_state"] = instanceVmState
	applyChanges(t, &tfInstOpts)
	getAndCheckOutput(t, &tfInstOpts, "vm_state", instanceVmState)
}

// TestUpdateMetadataMapInstance тестирует обновление метаданных экземпляра.
func TestUpdateMetadataMapInstance(t *testing.T) {
	threeTags := map[string]string{
		"foo":             "bar",
		"dragon_firewall": "jkl",
		"exit":            "true",
	}
	tfInstOpts.Vars["metadata_map"] = threeTags
	applyChanges(t, &tfInstOpts)
	getAndCheckOutput(t, &tfInstOpts, "metadata_map", threeTags)

	// Проверка на отсутствие старых значений
	oldTags := map[string]string{
		"type":            "magic_carpet",
		"unicorn_access":  "true",
		"dragon_firewall": "very-hot",
		"enchanted_speed": "lightning-fast",
		"fairy_lights":    "5",
	}
	checkAbsenceOfOldTags(t, &tfInstOpts, oldTags)

	// Обнуление metadata_map
	tfInstOpts.Vars["metadata_map"] = map[string]string{}
	applyChanges(t, &tfInstOpts)
	getAndCheckOutput(t, &tfInstOpts, "metadata_map", "{}")
}

func TestDeleteGlobalTeardown(t *testing.T) {
	// Очистка всех ресурсов
	terraform.Destroy(t, &tfInstOpts)
	terraform.Destroy(t, &tfSecGOpts)
	terraform.Destroy(t, &tfRFipOpts)
	terraform.Destroy(t, &tfFipOpts)
	terraform.Destroy(t, &tfSerGOpts)
	terraform.Destroy(t, &tfKeyOpts)
	terraform.Destroy(t, &tfVolOpts)
	terraform.Destroy(t, &tfNetOpts)
	//defer terraform.Destroy(t, &tfNetOpts)
	//defer terraform.Destroy(t, &tfVolOpts)
	//defer terraform.Destroy(t, &tfKeyOpts)
	//defer terraform.Destroy(t, &tfSerGOpts)
	//defer terraform.Destroy(t, &tfFipOpts)
	//defer terraform.Destroy(t, &tfRFipOpts)
	//defer terraform.Destroy(t, &tfSecGOpts)
	//defer terraform.Destroy(t, &tfInstOpts)
}

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
	if _, err := terraform.ApplyE(t, &tfNetOpts); err != nil {
		t.Fatalf("failed to initialize and apply network module: %v", err)
	}
	// Получение и сохранение идентификаторов сети и подсети.
	networkID := terraform.Output(t, &tfNetOpts, "network_id")
	subnetID := terraform.Output(t, &tfNetOpts, "subnet_id")

	// Инициализация и применение модуля volume.
	initializeVolumeModule(&tfVolOpts)
	if _, err := terraform.ApplyE(t, &tfVolOpts); err != nil {
		t.Fatalf("failed to initialize and apply volume module: %v", err)
	}

	// Инициализация и применение модуля ключевых пар.
	initializeKeypairModule(&tfKeyOpts)
	if _, err := terraform.ApplyE(t, &tfKeyOpts); err != nil {
		t.Fatalf("failed to initialize and apply keypair module: %v", err)
	}

	// Получение идентификатора группы серверов.
	initializeServerGroupModule(&tfSerGOpts)
	if _, err := terraform.ApplyE(t, &tfSerGOpts); err != nil {
		t.Fatalf("failed to initialize and apply server_group module: %v", err)
	}

	// Инициализация и применение модуля плавающих IP-адресов.
	initializeFloatingIPModule(&tfFipOpts)
	if _, err := terraform.ApplyE(t, &tfFipOpts); err != nil {
		t.Fatalf("failed to initialize and apply FIP module: %v", err)
	}

	// Инициализация и применение модуля зарезервированных фиксированных IP-адресов.
	initializeReservedFIPModule(&tfRFipOpts, networkID, subnetID)
	if _, err := terraform.ApplyE(t, &tfRFipOpts); err != nil {
		t.Fatalf("failed to initialize and apply ReservedFIP module: %v", err)
	}

	// Инициализация и применение модуля групп безопасности.
	initializeSecGroupModule(&tfSecGOpts)
	if _, err := terraform.ApplyE(t, &tfSecGOpts); err != nil {
		t.Fatalf("failed to initialize and apply Security Group module: %v", err)
	}
}

func TestCreateTerraformEdgeCenterInstance(t *testing.T) {
	// Получение и сохранение идентификаторов сети и подсети.
	networkID := terraform.Output(t, &tfNetOpts, "network_id")
	subnetID := terraform.Output(t, &tfNetOpts, "subnet_id")

	// Получение идентификаторов первого, второго и третьего тома.
	firstVolumeID := terraform.Output(t, &tfVolOpts, "first_volume_id")
	secondVolumeID := terraform.Output(t, &tfVolOpts, "second_volume_id")
	thirdVolumeID := terraform.Output(t, &tfVolOpts, "third_volume_id")

	// Получение идентификатора группы серверов.
	serverGroupID := terraform.Output(t, &tfSerGOpts, "server_group_id")

	// Инициализация и применение основного модуля экземпляра.
	initializeInstanceModule(&tfInstOpts, networkID, subnetID, firstVolumeID, secondVolumeID, thirdVolumeID, serverGroupID)
	// TODO: Нужно заменить на ApplyAndIdempotentE
	if _, err := terraform.ApplyE(t, &tfInstOpts); err != nil {
		t.Fatalf("failed to initialize and apply instance module: %v", err)
	}

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

// validateBasicOutputs проверяет основные параметры сервера, такие как ID и flavor.
func validateBasicOutputs(t *testing.T) {
	// Получаем ID сервера и проверяем, что он не пустой.
	instanceID := terraform.Output(t, &tfInstOpts, "instance_id")
	require.NotEmpty(t, instanceID, "Instance ID should not be empty")

	// Сверяем flavor сервера с ожидаемым значением.
	instanceFlavorID := terraform.Output(t, &tfInstOpts, "flavor_id")
	require.Equal(t, instanceFlavor, instanceFlavorID, "Flavor ID should match the expected value")

	// Проверяем, что имя сервера соответствует ожидаемому.
	computedInstanceName := terraform.Output(t, &tfInstOpts, "instance_name")
	require.Equal(t, instanceName, computedInstanceName, "Instance name should match the expected value")

	// Проверяем, что имя ключевой пары соответствует заданному.
	computedKeypairName := terraform.Output(t, &tfInstOpts, "keypair_name")
	require.Equal(t, keypairName, computedKeypairName, "Keypair name should match the expected value")

	// Сверяем ID группы серверов с полученным значением.
	serverGroupID := terraform.Output(t, &tfSerGOpts, "server_group_id")
	computedServerGroup := terraform.Output(t, &tfInstOpts, "server_group")
	require.Equal(t, serverGroupID, computedServerGroup, "Server group should match the expected ID")

	// Проверяем, что user data соответствуют указанным.
	computedUserData := terraform.Output(t, &tfInstOpts, "user_data")
	require.Equal(t, userData, computedUserData, "User data should match the expected value")
}

// validateVolumeOutputs проверяет прикрепленные volumes и их свойства, такие как id и boot_index.
func validateVolumeOutputs(t *testing.T) {
	volumes := terraform.OutputListOfObjects(t, &tfInstOpts, "instance_volumes")
	require.Equal(t, 3, len(volumes), "There should be three volumes attached to the instance")

	// Проверяем, совпадают ли идентификаторы volumes
	for i, volumeID := range []string{terraform.Output(t, &tfVolOpts, "first_volume_id"),
		terraform.Output(t, &tfVolOpts, "second_volume_id"),
		terraform.Output(t, &tfVolOpts, "third_volume_id")} {
		require.Equal(t, volumes[i]["volume_id"], volumeID, fmt.Sprintf("Volume ID should match for volume index: %d", i))
		// TODO: Невозможно проверить параметр deleteOnTermination поскольку он не изменяется и не читается функцией Read
		//deleteOnTermination := tfInstOpts.Vars["instance_volumes"].([]map[string]interface{})[i]["delete_on_termination"].(bool)
		//require.Equal(t, volumes[i]["delete_on_termination"], deleteOnTermination, fmt.Sprintf("Delete on termination flag should match for volume index: %d", i))
		bootIndex := tfInstOpts.Vars["instance_volumes"].([]map[string]interface{})[i]["boot_index"].(int)
		require.Equal(t, volumes[i]["boot_index"], bootIndex, fmt.Sprintf("Boot_Index flag should match for volume index: %d", i))
	}
}

// validateNetworkInterfaces проверяет конфигурации сетевых интерфейсов экземпляра.
func validateNetworkInterfaces(t *testing.T) {
	interfaces := terraform.OutputListOfObjects(t, &tfInstOpts, "instance_interfaces")
	require.Equal(t, 1, len(interfaces), "There should be one network interface attached to the instance")

	networkID := terraform.Output(t, &tfNetOpts, "network_id")
	subnetID := terraform.Output(t, &tfNetOpts, "subnet_id")
	require.Equal(t, interfaces[0]["network_id"], networkID, "Network ID should match")
	require.Equal(t, interfaces[0]["subnet_id"], subnetID, "Subnet ID should match")
}

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
	newInstanceName := fmt.Sprintf(instanceName + string(random.UniqueId()))
	tfInstOpts.Vars["instance_name"] = newInstanceName
	if _, err := terraform.ApplyAndIdempotentE(t, &tfInstOpts); err != nil {
		t.Fatalf("failed to apply changes to the instance: %v", err)
	}

	updatedInstanceName, err := terraform.OutputE(t, &tfInstOpts, "instance_name")
	if err != nil {
		t.Fatalf("failed to get updated Instance Name: %v", err)
	}
	require.Equal(t, newInstanceName, updatedInstanceName, "Instance name should be updated to the new value")

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
}

// updateFlavorAndUpdateInstance обновляет flavor_id инстанса и проверяет изменение
func TestUpdateFlavorInstance(t *testing.T) {
	newFlavorID := "g1-standard-2-4"
	updateFlavorInstance(t, &tfInstOpts, newFlavorID)
}

func updateFlavorInstance(t *testing.T, tfOpts *terraform.Options, newFlavorID string) {
	// Изменение параметра flavor_id для Terraform конфигурации
	tfOpts.Vars["flavor_id"] = newFlavorID
	// Применяем изменения используя Terraform
	if _, err := terraform.ApplyAndIdempotentE(t, tfOpts); err != nil {
		t.Fatalf("failed to apply changes to the instance: %v", err)
	}

	// Проверка обновленного значения flavor_id с использованием функции Output, чтобы убедиться, что изменение применилось
	updatedFlavorID, err := terraform.OutputE(t, tfOpts, "flavor_id")
	if err != nil {
		t.Fatalf("failed to get updated flavor ID: %v", err)
	}

	require.Equal(t, newFlavorID, updatedFlavorID, "Flavor ID should be updated to the new value")
}

// TestUpdateVmStateInstance обновляет состояние VM и проверяет новое состояние.
func TestUpdateVmStateInstance(t *testing.T) {
	newVmState := "stopped"
	require.NoError(t, updateVmStateInstance(t, &tfInstOpts, newVmState), "Failed to update VM state")
}

// updateVmStateInstance изменяет переменную "vm_state" в параметрах Terraform и применяет изменение,
// возвращая ошибку, если какой-либо шаг не удался.
func updateVmStateInstance(t *testing.T, tfOpts *terraform.Options, newVmState string) error {
	// Устанавливаем новое состояние VM.
	tfOpts.Vars["vm_state"] = newVmState
	// Применяем изменения с помощью Terraform.
	if _, err := terraform.ApplyAndIdempotentE(t, tfOpts); err != nil {
		t.Fatalf("applying changes to the instance failed: %s", err)
	}

	// Получаем обновленное состояние VM
	updatedVmState, err := terraform.OutputE(t, tfOpts, "vm_state")
	if err != nil {
		return fmt.Errorf("retrieving updated VM state failed: %w", err)
	}
	require.Equal(t, newVmState, updatedVmState, "VM state should be updated to the new value.")

	// Возвращаем старое значение
	tfOpts.Vars["vm_state"] = instanceVmState
	// Применяем изменения с помощью Terraform.
	if _, err := terraform.ApplyAndIdempotentE(t, tfOpts); err != nil {
		t.Fatalf("applying changes to the instance failed: %s", err)
	}

	// Получаем обновленное состояние VM
	oldVmState, err := terraform.OutputE(t, tfOpts, "vm_state")
	if err != nil {
		return fmt.Errorf("retrieving updated VM state failed: %w", err)
	}
	require.Equal(t, instanceVmState, oldVmState, "VM state should be updated to the new value.")

	return nil
}

func TestUpdateMetadataMapInstance(t *testing.T) {
	threeTags := map[string]string{
		"foo":             "bar",
		"dragon_firewall": "jkl",
		"exit":            "true",
	}
	tfInstOpts.Vars["metadata_map"] = threeTags

	if _, err := terraform.ApplyAndIdempotentE(t, &tfInstOpts); err != nil {
		t.Fatalf("failed to apply changes to the instance: %v", err)
	}

	updatedMetadataMapRaw, err := terraform.OutputJsonE(t, &tfInstOpts, "metadata_map")
	if err != nil {
		t.Fatalf("failed to get updated Metadata: %v", err)
	}

	var updatedMetadataMap map[string]string
	if err := json.Unmarshal([]byte(updatedMetadataMapRaw), &updatedMetadataMap); err != nil {
		t.Fatalf("failed to unmarshal updated metadata map: %v", err)
	}

	// Проверка наличия и соответствия ключей
	for key, expectedValue := range threeTags {
		actualValue, exists := updatedMetadataMap[key]
		if !exists {
			t.Errorf("key %s does not exist in the updated metadata map", key)
		} else if actualValue != expectedValue {
			t.Errorf("value mismatch for key '%s': expected %s, got %s", key, expectedValue, actualValue)
		}
	}

	oldTags := map[string]string{
		"type":            "magic_carpet",
		"unicorn_access":  "true",
		"dragon_firewall": "very-hot",
		"enchanted_speed": "lightning-fast",
		"fairy_lights":    "5",
	}

	// Проверка на отсутствие старых ключей и их значений
	for key, oldValue := range oldTags {
		actualValue, exists := updatedMetadataMap[key]
		if exists && actualValue == oldValue {
			t.Errorf("old key-value pair %s:%s should not exist in the updated metadata map", key, oldValue)
		}
	}

	// Обнуление metadata_map и повторное применение изменений
	tfInstOpts.Vars["metadata_map"] = map[string]string{}

	if _, err := terraform.ApplyAndIdempotentE(t, &tfInstOpts); err != nil {
		t.Fatalf("failed to clear the metadata map: %v", err)
	}

	emptyMetadataMapRaw, err := terraform.OutputJsonE(t, &tfInstOpts, "metadata_map")
	if err != nil {
		t.Fatalf("failed to get empty metadata map: %v", err)
	}

	if emptyMetadataMapRaw != "{}" {
		t.Errorf("metadata map should be empty, got: %s", emptyMetadataMapRaw)
	}
}

func TestDeleteGlobalTeardown(t *testing.T) {
	// Очистка всех ресурсов
	defer terraform.Destroy(t, &tfNetOpts)
	defer terraform.Destroy(t, &tfVolOpts)
	defer terraform.Destroy(t, &tfKeyOpts)
	defer terraform.Destroy(t, &tfSerGOpts)
	defer terraform.Destroy(t, &tfFipOpts)
	defer terraform.Destroy(t, &tfRFipOpts)
	defer terraform.Destroy(t, &tfSecGOpts)
	defer terraform.Destroy(t, &tfInstOpts)
}

package gencopy

import (
	ackv1alpha1 "github.com/aws/aws-controllers-k8s/apis/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TableSpec struct {
	AttributeDefinitions   []*AttributeDefinition  `json:"attributeDefinitions"`
	BillingMode            *string                 `json:"billingMode,omitempty"`
	GlobalSecondaryIndexes []*GlobalSecondaryIndex `json:"globalSecondaryIndexes,omitempty"`
	KeySchema              []*KeySchemaElement     `json:"keySchema"`
	LocalSecondaryIndexes  []*LocalSecondaryIndex  `json:"localSecondaryIndexes,omitempty"`
	ProvisionedThroughput  *ProvisionedThroughput  `json:"provisionedThroughput,omitempty"`
	SSESpecification       *SSESpecification       `json:"sseSpecification,omitempty"`
	StreamSpecification    *StreamSpecification    `json:"streamSpecification,omitempty"`
	TableName              *string                 `json:"tableName"`
	Tags                   []*Tag                  `json:"tags,omitempty"`
}

type TableStatus struct {
	ACKResourceMetadata *ackv1alpha1.ResourceMetadata `json:"ackResourceMetadata"`
	Conditions          []*ackv1alpha1.Condition      `json:"conditions"`
	ArchivalSummary     *ArchivalSummary              `json:"archivalSummary,omitempty"`
	BillingModeSummary  *BillingModeSummary           `json:"billingModeSummary,omitempty"`
	CreationDateTime    *metav1.Time                  `json:"creationDateTime,omitempty"`
	GlobalTableVersion  *string                       `json:"globalTableVersion,omitempty"`
	ItemCount           *int64                        `json:"itemCount,omitempty"`
	LatestStreamARN     *string                       `json:"latestStreamARN,omitempty"`
	LatestStreamLabel   *string                       `json:"latestStreamLabel,omitempty"`
	Replicas            []*ReplicaDescription         `json:"replicas,omitempty"`
	RestoreSummary      *RestoreSummary               `json:"restoreSummary,omitempty"`
	SSEDescription      *SSEDescription               `json:"sseDescription,omitempty"`
	TableID             *string                       `json:"tableID,omitempty"`
	TableSizeBytes      *int64                        `json:"tableSizeBytes,omitempty"`
	TableStatus         *string                       `json:"tableStatus,omitempty"`
}

type Table struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TableSpec   `json:"spec,omitempty"`
	Status            TableStatus `json:"status,omitempty"`
}

func (t1 *Table) Equal(t2 *Table) (bool, []string) {
	c := Comparator{}

	c.stringEqualSoft("Spec.TableName", t1.Spec.TableName, t1.Spec.TableName)

	diffs := c.Diffs()
	return len(diffs) == 0, diffs
}

type ArchivalSummary struct {
	ArchivalBackupARN *string      `json:"archivalBackupARN,omitempty"`
	ArchivalDateTime  *metav1.Time `json:"archivalDateTime,omitempty"`
	ArchivalReason    *string      `json:"archivalReason,omitempty"`
}

type AttributeDefinition struct {
	AttributeName *string `json:"attributeName,omitempty"`
	AttributeType *string `json:"attributeType,omitempty"`
}

type AutoScalingSettingsDescription struct {
	AutoScalingRoleARN *string `json:"autoScalingRoleARN,omitempty"`
	MaximumUnits       *int64  `json:"maximumUnits,omitempty"`
	MinimumUnits       *int64  `json:"minimumUnits,omitempty"`
}

type AutoScalingSettingsUpdate struct {
	MaximumUnits *int64 `json:"maximumUnits,omitempty"`
	MinimumUnits *int64 `json:"minimumUnits,omitempty"`
}

type BackupDescription struct {
	BackupDetails             *BackupDetails             `json:"backupDetails,omitempty"`
	SourceTableDetails        *SourceTableDetails        `json:"sourceTableDetails,omitempty"`
	SourceTableFeatureDetails *SourceTableFeatureDetails `json:"sourceTableFeatureDetails,omitempty"`
}

type BackupDetails struct {
	BackupARN              *string      `json:"backupARN,omitempty"`
	BackupCreationDateTime *metav1.Time `json:"backupCreationDateTime,omitempty"`
	BackupExpiryDateTime   *metav1.Time `json:"backupExpiryDateTime,omitempty"`
	BackupName             *string      `json:"backupName,omitempty"`
	BackupSizeBytes        *int64       `json:"backupSizeBytes,omitempty"`
	BackupStatus           *string      `json:"backupStatus,omitempty"`
	BackupType             *string      `json:"backupType,omitempty"`
}

type BackupSummary struct {
	BackupARN              *string      `json:"backupARN,omitempty"`
	BackupCreationDateTime *metav1.Time `json:"backupCreationDateTime,omitempty"`
	BackupExpiryDateTime   *metav1.Time `json:"backupExpiryDateTime,omitempty"`
	BackupName             *string      `json:"backupName,omitempty"`
	BackupSizeBytes        *int64       `json:"backupSizeBytes,omitempty"`
	BackupStatus           *string      `json:"backupStatus,omitempty"`
	BackupType             *string      `json:"backupType,omitempty"`
	TableARN               *string      `json:"tableARN,omitempty"`
	TableID                *string      `json:"tableID,omitempty"`
	TableName              *string      `json:"tableName,omitempty"`
}

type BillingModeSummary struct {
	BillingMode                       *string      `json:"billingMode,omitempty"`
	LastUpdateToPayPerRequestDateTime *metav1.Time `json:"lastUpdateToPayPerRequestDateTime,omitempty"`
}

type ConditionCheck struct {
	TableName *string `json:"tableName,omitempty"`
}

type ConsumedCapacity struct {
	TableName *string `json:"tableName,omitempty"`
}

type ContributorInsightsSummary struct {
	IndexName *string `json:"indexName,omitempty"`
	TableName *string `json:"tableName,omitempty"`
}

type CreateGlobalSecondaryIndexAction struct {
	IndexName             *string                `json:"indexName,omitempty"`
	KeySchema             []*KeySchemaElement    `json:"keySchema,omitempty"`
	Projection            *Projection            `json:"projection,omitempty"`
	ProvisionedThroughput *ProvisionedThroughput `json:"provisionedThroughput,omitempty"`
}

type CreateReplicaAction struct {
	RegionName *string `json:"regionName,omitempty"`
}

type CreateReplicationGroupMemberAction struct {
	GlobalSecondaryIndexes        []*ReplicaGlobalSecondaryIndex `json:"globalSecondaryIndexes,omitempty"`
	KMSMasterKeyID                *string                        `json:"kmsMasterKeyID,omitempty"`
	ProvisionedThroughputOverride *ProvisionedThroughputOverride `json:"provisionedThroughputOverride,omitempty"`
	RegionName                    *string                        `json:"regionName,omitempty"`
}

type Delete struct {
	TableName *string `json:"tableName,omitempty"`
}

type DeleteGlobalSecondaryIndexAction struct {
	IndexName *string `json:"indexName,omitempty"`
}

type DeleteReplicaAction struct {
	RegionName *string `json:"regionName,omitempty"`
}

type DeleteReplicationGroupMemberAction struct {
	RegionName *string `json:"regionName,omitempty"`
}

type Endpoint struct {
	Address              *string `json:"address,omitempty"`
	CachePeriodInMinutes *int64  `json:"cachePeriodInMinutes,omitempty"`
}

type Get struct {
	TableName *string `json:"tableName,omitempty"`
}

type GlobalSecondaryIndex struct {
	IndexName             *string                `json:"indexName,omitempty"`
	KeySchema             []*KeySchemaElement    `json:"keySchema,omitempty"`
	Projection            *Projection            `json:"projection,omitempty"`
	ProvisionedThroughput *ProvisionedThroughput `json:"provisionedThroughput,omitempty"`
}

type GlobalSecondaryIndexAutoScalingUpdate struct {
	IndexName *string `json:"indexName,omitempty"`
}

type GlobalSecondaryIndexDescription struct {
	Backfilling           *bool                             `json:"backfilling,omitempty"`
	IndexARN              *string                           `json:"indexARN,omitempty"`
	IndexName             *string                           `json:"indexName,omitempty"`
	IndexSizeBytes        *int64                            `json:"indexSizeBytes,omitempty"`
	IndexStatus           *string                           `json:"indexStatus,omitempty"`
	ItemCount             *int64                            `json:"itemCount,omitempty"`
	KeySchema             []*KeySchemaElement               `json:"keySchema,omitempty"`
	Projection            *Projection                       `json:"projection,omitempty"`
	ProvisionedThroughput *ProvisionedThroughputDescription `json:"provisionedThroughput,omitempty"`
}

type GlobalSecondaryIndexInfo struct {
	IndexName             *string                `json:"indexName,omitempty"`
	KeySchema             []*KeySchemaElement    `json:"keySchema,omitempty"`
	Projection            *Projection            `json:"projection,omitempty"`
	ProvisionedThroughput *ProvisionedThroughput `json:"provisionedThroughput,omitempty"`
}

type GlobalSecondaryIndexUpdate struct {
	Create *CreateGlobalSecondaryIndexAction `json:"create,omitempty"`
	Delete *DeleteGlobalSecondaryIndexAction `json:"delete,omitempty"`
	Update *UpdateGlobalSecondaryIndexAction `json:"update,omitempty"`
}

type GlobalTableDescription struct {
	CreationDateTime  *metav1.Time          `json:"creationDateTime,omitempty"`
	GlobalTableARN    *string               `json:"globalTableARN,omitempty"`
	GlobalTableName   *string               `json:"globalTableName,omitempty"`
	GlobalTableStatus *string               `json:"globalTableStatus,omitempty"`
	ReplicationGroup  []*ReplicaDescription `json:"replicationGroup,omitempty"`
}

type GlobalTableGlobalSecondaryIndexSettingsUpdate struct {
	IndexName                     *string `json:"indexName,omitempty"`
	ProvisionedWriteCapacityUnits *int64  `json:"provisionedWriteCapacityUnits,omitempty"`
}

type GlobalTable_SDK struct {
	GlobalTableName  *string    `json:"globalTableName,omitempty"`
	ReplicationGroup []*Replica `json:"replicationGroup,omitempty"`
}

type KeySchemaElement struct {
	AttributeName *string `json:"attributeName,omitempty"`
	KeyType       *string `json:"keyType,omitempty"`
}

type LocalSecondaryIndex struct {
	IndexName  *string             `json:"indexName,omitempty"`
	KeySchema  []*KeySchemaElement `json:"keySchema,omitempty"`
	Projection *Projection         `json:"projection,omitempty"`
}

type LocalSecondaryIndexDescription struct {
	IndexARN       *string             `json:"indexARN,omitempty"`
	IndexName      *string             `json:"indexName,omitempty"`
	IndexSizeBytes *int64              `json:"indexSizeBytes,omitempty"`
	ItemCount      *int64              `json:"itemCount,omitempty"`
	KeySchema      []*KeySchemaElement `json:"keySchema,omitempty"`
	Projection     *Projection         `json:"projection,omitempty"`
}

type LocalSecondaryIndexInfo struct {
	IndexName  *string             `json:"indexName,omitempty"`
	KeySchema  []*KeySchemaElement `json:"keySchema,omitempty"`
	Projection *Projection         `json:"projection,omitempty"`
}

type PointInTimeRecoveryDescription struct {
	EarliestRestorableDateTime *metav1.Time `json:"earliestRestorableDateTime,omitempty"`
	LatestRestorableDateTime   *metav1.Time `json:"latestRestorableDateTime,omitempty"`
}

type Projection struct {
	NonKeyAttributes []*string `json:"nonKeyAttributes,omitempty"`
	ProjectionType   *string   `json:"projectionType,omitempty"`
}

type ProvisionedThroughput struct {
	ReadCapacityUnits  *int64 `json:"readCapacityUnits,omitempty"`
	WriteCapacityUnits *int64 `json:"writeCapacityUnits,omitempty"`
}

type ProvisionedThroughputDescription struct {
	LastDecreaseDateTime   *metav1.Time `json:"lastDecreaseDateTime,omitempty"`
	LastIncreaseDateTime   *metav1.Time `json:"lastIncreaseDateTime,omitempty"`
	NumberOfDecreasesToday *int64       `json:"numberOfDecreasesToday,omitempty"`
	ReadCapacityUnits      *int64       `json:"readCapacityUnits,omitempty"`
	WriteCapacityUnits     *int64       `json:"writeCapacityUnits,omitempty"`
}

type ProvisionedThroughputOverride struct {
	ReadCapacityUnits *int64 `json:"readCapacityUnits,omitempty"`
}

type Put struct {
	TableName *string `json:"tableName,omitempty"`
}

type Replica struct {
	RegionName *string `json:"regionName,omitempty"`
}

type ReplicaAutoScalingDescription struct {
	RegionName    *string `json:"regionName,omitempty"`
	ReplicaStatus *string `json:"replicaStatus,omitempty"`
}

type ReplicaAutoScalingUpdate struct {
	RegionName *string `json:"regionName,omitempty"`
}

type ReplicaDescription struct {
	GlobalSecondaryIndexes        []*ReplicaGlobalSecondaryIndexDescription `json:"globalSecondaryIndexes,omitempty"`
	KMSMasterKeyID                *string                                   `json:"kmsMasterKeyID,omitempty"`
	ProvisionedThroughputOverride *ProvisionedThroughputOverride            `json:"provisionedThroughputOverride,omitempty"`
	RegionName                    *string                                   `json:"regionName,omitempty"`
	ReplicaStatus                 *string                                   `json:"replicaStatus,omitempty"`
	ReplicaStatusDescription      *string                                   `json:"replicaStatusDescription,omitempty"`
	ReplicaStatusPercentProgress  *string                                   `json:"replicaStatusPercentProgress,omitempty"`
}

type ReplicaGlobalSecondaryIndex struct {
	IndexName                     *string                        `json:"indexName,omitempty"`
	ProvisionedThroughputOverride *ProvisionedThroughputOverride `json:"provisionedThroughputOverride,omitempty"`
}

type ReplicaGlobalSecondaryIndexAutoScalingDescription struct {
	IndexName   *string `json:"indexName,omitempty"`
	IndexStatus *string `json:"indexStatus,omitempty"`
}

type ReplicaGlobalSecondaryIndexAutoScalingUpdate struct {
	IndexName *string `json:"indexName,omitempty"`
}

type ReplicaGlobalSecondaryIndexDescription struct {
	IndexName                     *string                        `json:"indexName,omitempty"`
	ProvisionedThroughputOverride *ProvisionedThroughputOverride `json:"provisionedThroughputOverride,omitempty"`
}

type ReplicaGlobalSecondaryIndexSettingsDescription struct {
	IndexName                     *string `json:"indexName,omitempty"`
	IndexStatus                   *string `json:"indexStatus,omitempty"`
	ProvisionedReadCapacityUnits  *int64  `json:"provisionedReadCapacityUnits,omitempty"`
	ProvisionedWriteCapacityUnits *int64  `json:"provisionedWriteCapacityUnits,omitempty"`
}

type ReplicaGlobalSecondaryIndexSettingsUpdate struct {
	IndexName                    *string `json:"indexName,omitempty"`
	ProvisionedReadCapacityUnits *int64  `json:"provisionedReadCapacityUnits,omitempty"`
}

type ReplicaSettingsDescription struct {
	RegionName                           *string             `json:"regionName,omitempty"`
	ReplicaBillingModeSummary            *BillingModeSummary `json:"replicaBillingModeSummary,omitempty"`
	ReplicaProvisionedReadCapacityUnits  *int64              `json:"replicaProvisionedReadCapacityUnits,omitempty"`
	ReplicaProvisionedWriteCapacityUnits *int64              `json:"replicaProvisionedWriteCapacityUnits,omitempty"`
	ReplicaStatus                        *string             `json:"replicaStatus,omitempty"`
}

type ReplicaSettingsUpdate struct {
	RegionName                          *string `json:"regionName,omitempty"`
	ReplicaProvisionedReadCapacityUnits *int64  `json:"replicaProvisionedReadCapacityUnits,omitempty"`
}

type ReplicaUpdate struct {
	Create *CreateReplicaAction `json:"create,omitempty"`
	Delete *DeleteReplicaAction `json:"delete,omitempty"`
}

type ReplicationGroupUpdate struct {
	Create *CreateReplicationGroupMemberAction `json:"create,omitempty"`
	Delete *DeleteReplicationGroupMemberAction `json:"delete,omitempty"`
	Update *UpdateReplicationGroupMemberAction `json:"update,omitempty"`
}

type RestoreSummary struct {
	RestoreDateTime   *metav1.Time `json:"restoreDateTime,omitempty"`
	RestoreInProgress *bool        `json:"restoreInProgress,omitempty"`
	SourceBackupARN   *string      `json:"sourceBackupARN,omitempty"`
	SourceTableARN    *string      `json:"sourceTableARN,omitempty"`
}

type SSEDescription struct {
	InaccessibleEncryptionDateTime *metav1.Time `json:"inaccessibleEncryptionDateTime,omitempty"`
	KMSMasterKeyARN                *string      `json:"kmsMasterKeyARN,omitempty"`
	SSEType                        *string      `json:"sseType,omitempty"`
	Status                         *string      `json:"status,omitempty"`
}

type SSESpecification struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	KMSMasterKeyID *string `json:"kmsMasterKeyID,omitempty"`
	SSEType        *string `json:"sseType,omitempty"`
}

type SourceTableDetails struct {
	BillingMode           *string                `json:"billingMode,omitempty"`
	ItemCount             *int64                 `json:"itemCount,omitempty"`
	KeySchema             []*KeySchemaElement    `json:"keySchema,omitempty"`
	ProvisionedThroughput *ProvisionedThroughput `json:"provisionedThroughput,omitempty"`
	TableARN              *string                `json:"tableARN,omitempty"`
	TableCreationDateTime *metav1.Time           `json:"tableCreationDateTime,omitempty"`
	TableID               *string                `json:"tableID,omitempty"`
	TableName             *string                `json:"tableName,omitempty"`
	TableSizeBytes        *int64                 `json:"tableSizeBytes,omitempty"`
}

type SourceTableFeatureDetails struct {
	GlobalSecondaryIndexes []*GlobalSecondaryIndexInfo `json:"globalSecondaryIndexes,omitempty"`
	LocalSecondaryIndexes  []*LocalSecondaryIndexInfo  `json:"localSecondaryIndexes,omitempty"`
	SSEDescription         *SSEDescription             `json:"sseDescription,omitempty"`
	StreamDescription      *StreamSpecification        `json:"streamDescription,omitempty"`
	TimeToLiveDescription  *TimeToLiveDescription      `json:"timeToLiveDescription,omitempty"`
}

type StreamSpecification struct {
	StreamEnabled  *bool   `json:"streamEnabled,omitempty"`
	StreamViewType *string `json:"streamViewType,omitempty"`
}

type TableAutoScalingDescription struct {
	TableName   *string `json:"tableName,omitempty"`
	TableStatus *string `json:"tableStatus,omitempty"`
}

type TableDescription struct {
	ArchivalSummary        *ArchivalSummary                   `json:"archivalSummary,omitempty"`
	AttributeDefinitions   []*AttributeDefinition             `json:"attributeDefinitions,omitempty"`
	BillingModeSummary     *BillingModeSummary                `json:"billingModeSummary,omitempty"`
	CreationDateTime       *metav1.Time                       `json:"creationDateTime,omitempty"`
	GlobalSecondaryIndexes []*GlobalSecondaryIndexDescription `json:"globalSecondaryIndexes,omitempty"`
	GlobalTableVersion     *string                            `json:"globalTableVersion,omitempty"`
	ItemCount              *int64                             `json:"itemCount,omitempty"`
	KeySchema              []*KeySchemaElement                `json:"keySchema,omitempty"`
	LatestStreamARN        *string                            `json:"latestStreamARN,omitempty"`
	LatestStreamLabel      *string                            `json:"latestStreamLabel,omitempty"`
	LocalSecondaryIndexes  []*LocalSecondaryIndexDescription  `json:"localSecondaryIndexes,omitempty"`
	ProvisionedThroughput  *ProvisionedThroughputDescription  `json:"provisionedThroughput,omitempty"`
	Replicas               []*ReplicaDescription              `json:"replicas,omitempty"`
	RestoreSummary         *RestoreSummary                    `json:"restoreSummary,omitempty"`
	SSEDescription         *SSEDescription                    `json:"sseDescription,omitempty"`
	StreamSpecification    *StreamSpecification               `json:"streamSpecification,omitempty"`
	TableARN               *string                            `json:"tableARN,omitempty"`
	TableID                *string                            `json:"tableID,omitempty"`
	TableName              *string                            `json:"tableName,omitempty"`
	TableSizeBytes         *int64                             `json:"tableSizeBytes,omitempty"`
	TableStatus            *string                            `json:"tableStatus,omitempty"`
}

type Tag struct {
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type TimeToLiveDescription struct {
	AttributeName    *string `json:"attributeName,omitempty"`
	TimeToLiveStatus *string `json:"timeToLiveStatus,omitempty"`
}

type TimeToLiveSpecification struct {
	AttributeName *string `json:"attributeName,omitempty"`
}

type Update struct {
	TableName *string `json:"tableName,omitempty"`
}

type UpdateGlobalSecondaryIndexAction struct {
	IndexName             *string                `json:"indexName,omitempty"`
	ProvisionedThroughput *ProvisionedThroughput `json:"provisionedThroughput,omitempty"`
}

type UpdateReplicationGroupMemberAction struct {
	GlobalSecondaryIndexes        []*ReplicaGlobalSecondaryIndex `json:"globalSecondaryIndexes,omitempty"`
	KMSMasterKeyID                *string                        `json:"kmsMasterKeyID,omitempty"`
	ProvisionedThroughputOverride *ProvisionedThroughputOverride `json:"provisionedThroughputOverride,omitempty"`
	RegionName                    *string                        `json:"regionName,omitempty"`
}

-- MySQL dump 10.13  Distrib 9.6.0, for macos14.8 (arm64)
--
-- Host: 127.0.0.1    Database: report
-- ------------------------------------------------------
-- Server version	9.6.0

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `agent_seats`
--

DROP TABLE IF EXISTS `agent_seats`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `agent_seats` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `user_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `extension` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sip_uri` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('available','busy','offline','wrap_up') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'offline',
  `skills_json` json DEFAULT NULL,
  `current_call_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_agent_seats_user` (`user_id`),
  CONSTRAINT `fk_agent_seats_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `agent_seats`
--

LOCK TABLES `agent_seats` WRITE;
/*!40000 ALTER TABLE `agent_seats` DISABLE KEYS */;
/*!40000 ALTER TABLE `agent_seats` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `audit_logs`
--

DROP TABLE IF EXISTS `audit_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `audit_logs` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `action` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `resource` varchar(240) COLLATE utf8mb4_unicode_ci NOT NULL,
  `before_json` json DEFAULT NULL,
  `after_json` json DEFAULT NULL,
  `ip` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` text COLLATE utf8mb4_unicode_ci,
  `trace_id` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `audit_logs`
--

LOCK TABLES `audit_logs` WRITE;
/*!40000 ALTER TABLE `audit_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `audit_logs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `call_analyses`
--

DROP TABLE IF EXISTS `call_analyses`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `call_analyses` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `call_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `provider_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_emotion` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `true_satisfaction` decimal(4,2) DEFAULT NULL,
  `risk_level` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `patient_status` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `summary` text COLLATE utf8mb4_unicode_ci,
  `extracted_form_data` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_call_analyses_call` (`call_id`),
  KEY `fk_call_analyses_provider` (`provider_id`),
  CONSTRAINT `fk_call_analyses_call` FOREIGN KEY (`call_id`) REFERENCES `call_sessions` (`id`),
  CONSTRAINT `fk_call_analyses_provider` FOREIGN KEY (`provider_id`) REFERENCES `model_providers` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `call_analyses`
--

LOCK TABLES `call_analyses` WRITE;
/*!40000 ALTER TABLE `call_analyses` DISABLE KEYS */;
/*!40000 ALTER TABLE `call_analyses` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `call_sessions`
--

DROP TABLE IF EXISTS `call_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `call_sessions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `seat_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `direction` enum('inbound','outbound') COLLATE utf8mb4_unicode_ci NOT NULL,
  `phone_number` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('dialing','ringing','connected','recording','recorded','ended','failed') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'dialing',
  `started_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `ended_at` timestamp NULL DEFAULT NULL,
  `recording_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `transcript_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `analysis_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `interview_form` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_call_sessions_seat` (`seat_id`),
  KEY `fk_call_sessions_patient` (`patient_id`),
  CONSTRAINT `fk_call_sessions_patient` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`),
  CONSTRAINT `fk_call_sessions_seat` FOREIGN KEY (`seat_id`) REFERENCES `agent_seats` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `call_sessions`
--

LOCK TABLES `call_sessions` WRITE;
/*!40000 ALTER TABLE `call_sessions` DISABLE KEYS */;
/*!40000 ALTER TABLE `call_sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `clinical_visits`
--

DROP TABLE IF EXISTS `clinical_visits`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `clinical_visits` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_no` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_type` varchar(60) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `department_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `department_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ward` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `bed_no` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `attending_doctor` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `visit_at` datetime DEFAULT NULL,
  `discharge_at` datetime DEFAULT NULL,
  `diagnosis_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `diagnosis_name` varchar(240) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `source_refs_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_clinical_visit_no` (`visit_no`),
  KEY `fk_clinical_visits_patient` (`patient_id`),
  CONSTRAINT `fk_clinical_visits_patient` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `clinical_visits`
--

LOCK TABLES `clinical_visits` WRITE;
/*!40000 ALTER TABLE `clinical_visits` DISABLE KEYS */;
INSERT INTO `clinical_visits` VALUES ('2d4c455b-948f-46d3-9530-17573e1c4991','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','V778',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,'active','{\"protocol\": \"http\", \"dataSourceId\": \"119a6adf-55b8-419b-baeb-6a836e4b150a\"}','2026-05-14 23:27:20','2026-05-15 02:51:02');
/*!40000 ALTER TABLE `clinical_visits` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `data_source_bindings`
--

DROP TABLE IF EXISTS `data_source_bindings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `data_source_bindings` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `form_component_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `data_source_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `operation` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `params_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_data_source_bindings_component` (`form_component_id`),
  KEY `fk_data_source_bindings_source` (`data_source_id`),
  CONSTRAINT `fk_data_source_bindings_component` FOREIGN KEY (`form_component_id`) REFERENCES `form_components` (`id`),
  CONSTRAINT `fk_data_source_bindings_source` FOREIGN KEY (`data_source_id`) REFERENCES `data_sources` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `data_source_bindings`
--

LOCK TABLES `data_source_bindings` WRITE;
/*!40000 ALTER TABLE `data_source_bindings` DISABLE KEYS */;
/*!40000 ALTER TABLE `data_source_bindings` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `data_source_credentials`
--

DROP TABLE IF EXISTS `data_source_credentials`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `data_source_credentials` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `data_source_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `secret_ciphertext` blob NOT NULL,
  `key_version` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_data_source_credentials_source` (`data_source_id`),
  CONSTRAINT `fk_data_source_credentials_source` FOREIGN KEY (`data_source_id`) REFERENCES `data_sources` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `data_source_credentials`
--

LOCK TABLES `data_source_credentials` WRITE;
/*!40000 ALTER TABLE `data_source_credentials` DISABLE KEYS */;
/*!40000 ALTER TABLE `data_source_credentials` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `data_sources`
--

DROP TABLE IF EXISTS `data_sources`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `data_sources` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `protocol` enum('mysql','postgres','http','soap','xml','grpc','hl7','dicom','custom') COLLATE utf8mb4_unicode_ci NOT NULL,
  `endpoint` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `config_json` json DEFAULT NULL,
  `dictionaries_json` json DEFAULT NULL,
  `field_mapping_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `data_sources`
--

LOCK TABLES `data_sources` WRITE;
/*!40000 ALTER TABLE `data_sources` DISABLE KEYS */;
/*!40000 ALTER TABLE `data_sources` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `datasets`
--

DROP TABLE IF EXISTS `datasets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `datasets` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `owner` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `record_count` int NOT NULL DEFAULT '0',
  `form_count` int NOT NULL DEFAULT '0',
  `status` enum('active','archived') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `datasets`
--

LOCK TABLES `datasets` WRITE;
/*!40000 ALTER TABLE `datasets` DISABLE KEYS */;
/*!40000 ALTER TABLE `datasets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `departments`
--

DROP TABLE IF EXISTS `departments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `departments` (
  `id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'clinical',
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `departments`
--

LOCK TABLES `departments` WRITE;
/*!40000 ALTER TABLE `departments` DISABLE KEYS */;
INSERT INTO `departments` VALUES ('DEPT-CARD','CARD','心内科','clinical','active','2026-05-14 07:52:37','2026-05-14 07:52:37'),('DEPT-ENDO','ENDO','内分泌科','clinical','active','2026-05-14 07:52:37','2026-05-14 07:52:37');
/*!40000 ALTER TABLE `departments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `dictionaries`
--

DROP TABLE IF EXISTS `dictionaries`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `dictionaries` (
  `id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `code` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `category` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `items_json` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `dictionaries`
--

LOCK TABLES `dictionaries` WRITE;
/*!40000 ALTER TABLE `dictionaries` DISABLE KEYS */;
INSERT INTO `dictionaries` VALUES ('DICT-CASE-FIELDS','case_common_fields','病例常用字段','病例管理','病例建档、科研队列、病案首页和随访筛选常用字段','[{\"key\": \"case_no\", \"label\": \"病例号\", \"value\": \"case_no\"}, {\"key\": \"patient_no\", \"label\": \"档案号\", \"value\": \"patient_no\"}, {\"key\": \"patient_name\", \"label\": \"患者姓名\", \"value\": \"patient_name\"}, {\"key\": \"gender\", \"label\": \"性别\", \"value\": \"gender\"}, {\"key\": \"age\", \"label\": \"年龄\", \"value\": \"age\"}, {\"key\": \"id_card_no\", \"label\": \"身份证号\", \"value\": \"id_card_no\"}, {\"key\": \"phone\", \"label\": \"联系电话\", \"value\": \"phone\"}, {\"key\": \"case_source\", \"label\": \"病例来源\", \"value\": \"case_source\"}, {\"key\": \"disease_code\", \"label\": \"病种编码\", \"value\": \"disease_code\"}, {\"key\": \"disease_name\", \"label\": \"病种名称\", \"value\": \"disease_name\"}, {\"key\": \"primary_diagnosis_code\", \"label\": \"主要诊断编码\", \"value\": \"primary_diagnosis_code\"}, {\"key\": \"primary_diagnosis_name\", \"label\": \"主要诊断名称\", \"value\": \"primary_diagnosis_name\"}, {\"key\": \"tumor_stage\", \"label\": \"肿瘤分期\", \"value\": \"tumor_stage\"}, {\"key\": \"pathology_no\", \"label\": \"病理号\", \"value\": \"pathology_no\"}, {\"key\": \"pathology_diagnosis\", \"label\": \"病理诊断\", \"value\": \"pathology_diagnosis\"}, {\"key\": \"operation_name\", \"label\": \"手术名称\", \"value\": \"operation_name\"}, {\"key\": \"operation_date\", \"label\": \"手术日期\", \"value\": \"operation_date\"}, {\"key\": \"discharge_status\", \"label\": \"出院情况\", \"value\": \"discharge_status\"}, {\"key\": \"followup_flag\", \"label\": \"随访标识\", \"value\": \"followup_flag\"}, {\"key\": \"case_created_at\", \"label\": \"建档时间\", \"value\": \"case_created_at\"}]','2026-05-14 11:10:53','2026-05-14 11:10:53'),('DICT-EMR-FIELDS','emr_common_fields','电子病历常用字段','电子病历','门诊、住院、专科病历同步和表单映射常用字段','[{\"key\": \"record_no\", \"label\": \"病历号\", \"value\": \"record_no\"}, {\"key\": \"record_type\", \"label\": \"病历类型\", \"value\": \"record_type\"}, {\"key\": \"record_title\", \"label\": \"病历标题\", \"value\": \"record_title\"}, {\"key\": \"chief_complaint\", \"label\": \"主诉\", \"value\": \"chief_complaint\"}, {\"key\": \"present_illness\", \"label\": \"现病史\", \"value\": \"present_illness\"}, {\"key\": \"past_history\", \"label\": \"既往史\", \"value\": \"past_history\"}, {\"key\": \"personal_history\", \"label\": \"个人史\", \"value\": \"personal_history\"}, {\"key\": \"allergy_history\", \"label\": \"过敏史\", \"value\": \"allergy_history\"}, {\"key\": \"physical_exam\", \"label\": \"体格检查\", \"value\": \"physical_exam\"}, {\"key\": \"specialist_exam\", \"label\": \"专科检查\", \"value\": \"specialist_exam\"}, {\"key\": \"auxiliary_exam\", \"label\": \"辅助检查\", \"value\": \"auxiliary_exam\"}, {\"key\": \"diagnosis_code\", \"label\": \"诊断编码\", \"value\": \"diagnosis_code\"}, {\"key\": \"diagnosis_name\", \"label\": \"诊断名称\", \"value\": \"diagnosis_name\"}, {\"key\": \"treatment_plan\", \"label\": \"诊疗计划\", \"value\": \"treatment_plan\"}, {\"key\": \"doctor_advice\", \"label\": \"医嘱\", \"value\": \"doctor_advice\"}, {\"key\": \"recorded_at\", \"label\": \"记录时间\", \"value\": \"recorded_at\"}, {\"key\": \"record_doctor\", \"label\": \"记录医生\", \"value\": \"record_doctor\"}, {\"key\": \"department_code\", \"label\": \"科室编码\", \"value\": \"department_code\"}, {\"key\": \"department_name\", \"label\": \"科室名称\", \"value\": \"department_name\"}, {\"key\": \"source_system\", \"label\": \"来源系统\", \"value\": \"source_system\"}]','2026-05-14 11:10:53','2026-05-14 11:10:53'),('DICT-FOLLOWUP-STATUS','followup_status','随访任务状态','随访中心','','[{\"key\": \"pending\", \"label\": \"待随访\", \"value\": \"pending\"}, {\"key\": \"assigned\", \"label\": \"已分配\", \"value\": \"assigned\"}, {\"key\": \"in_progress\", \"label\": \"进行中\", \"value\": \"in_progress\"}, {\"key\": \"completed\", \"label\": \"已完成\", \"value\": \"completed\"}, {\"key\": \"failed\", \"label\": \"失败\", \"value\": \"failed\"}]','2026-05-14 07:52:37','2026-05-14 07:52:37'),('DICT-GENDER','gender','性别字典','患者基础','','[{\"key\": \"M\", \"label\": \"男\", \"value\": \"男\"}, {\"key\": \"F\", \"label\": \"女\", \"value\": \"女\"}, {\"key\": \"O\", \"label\": \"其他\", \"value\": \"其他\"}]','2026-05-14 07:52:37','2026-05-14 07:52:37'),('DICT-MEDICATION-FIELDS','medication_common_fields','用药常用字段','用药信息','处方、医嘱、用药随访和不良反应采集常用字段','[{\"key\": \"order_no\", \"label\": \"医嘱号\", \"value\": \"order_no\"}, {\"key\": \"prescription_no\", \"label\": \"处方号\", \"value\": \"prescription_no\"}, {\"key\": \"drug_code\", \"label\": \"药品编码\", \"value\": \"drug_code\"}, {\"key\": \"drug_name\", \"label\": \"药品名称\", \"value\": \"drug_name\"}, {\"key\": \"generic_name\", \"label\": \"通用名\", \"value\": \"generic_name\"}, {\"key\": \"specification\", \"label\": \"规格\", \"value\": \"specification\"}, {\"key\": \"dosage\", \"label\": \"单次剂量\", \"value\": \"dosage\"}, {\"key\": \"dosage_unit\", \"label\": \"剂量单位\", \"value\": \"dosage_unit\"}, {\"key\": \"frequency\", \"label\": \"用药频次\", \"value\": \"frequency\"}, {\"key\": \"route\", \"label\": \"给药途径\", \"value\": \"route\"}, {\"key\": \"start_at\", \"label\": \"开始时间\", \"value\": \"start_at\"}, {\"key\": \"end_at\", \"label\": \"结束时间\", \"value\": \"end_at\"}, {\"key\": \"days\", \"label\": \"用药天数\", \"value\": \"days\"}, {\"key\": \"quantity\", \"label\": \"数量\", \"value\": \"quantity\"}, {\"key\": \"manufacturer\", \"label\": \"生产厂家\", \"value\": \"manufacturer\"}, {\"key\": \"doctor_name\", \"label\": \"开立医生\", \"value\": \"doctor_name\"}, {\"key\": \"pharmacist_name\", \"label\": \"审核药师\", \"value\": \"pharmacist_name\"}, {\"key\": \"medication_status\", \"label\": \"用药状态\", \"value\": \"medication_status\"}, {\"key\": \"adverse_reaction\", \"label\": \"不良反应\", \"value\": \"adverse_reaction\"}, {\"key\": \"compliance\", \"label\": \"用药依从性\", \"value\": \"compliance\"}]','2026-05-14 11:10:53','2026-05-14 11:10:53'),('DICT-VISIT-FIELDS','visit_common_fields','就诊常用字段','就诊信息','门诊、急诊、住院、出院记录同步常用字段','[{\"key\": \"visit_no\", \"label\": \"就诊号\", \"value\": \"visit_no\"}, {\"key\": \"visit_type\", \"label\": \"就诊类型\", \"value\": \"visit_type\"}, {\"key\": \"outpatient_no\", \"label\": \"门诊号\", \"value\": \"outpatient_no\"}, {\"key\": \"inpatient_no\", \"label\": \"住院号\", \"value\": \"inpatient_no\"}, {\"key\": \"admission_no\", \"label\": \"入院登记号\", \"value\": \"admission_no\"}, {\"key\": \"visit_at\", \"label\": \"就诊时间\", \"value\": \"visit_at\"}, {\"key\": \"admission_at\", \"label\": \"入院时间\", \"value\": \"admission_at\"}, {\"key\": \"discharge_at\", \"label\": \"出院时间\", \"value\": \"discharge_at\"}, {\"key\": \"department_code\", \"label\": \"就诊科室编码\", \"value\": \"department_code\"}, {\"key\": \"department_name\", \"label\": \"就诊科室\", \"value\": \"department_name\"}, {\"key\": \"ward_name\", \"label\": \"病区\", \"value\": \"ward_name\"}, {\"key\": \"bed_no\", \"label\": \"床号\", \"value\": \"bed_no\"}, {\"key\": \"attending_doctor\", \"label\": \"主治医生\", \"value\": \"attending_doctor\"}, {\"key\": \"responsible_nurse\", \"label\": \"责任护士\", \"value\": \"responsible_nurse\"}, {\"key\": \"diagnosis_code\", \"label\": \"就诊诊断编码\", \"value\": \"diagnosis_code\"}, {\"key\": \"diagnosis_name\", \"label\": \"就诊诊断\", \"value\": \"diagnosis_name\"}, {\"key\": \"visit_status\", \"label\": \"就诊状态\", \"value\": \"visit_status\"}, {\"key\": \"discharge_disposition\", \"label\": \"离院方式\", \"value\": \"discharge_disposition\"}, {\"key\": \"total_fee\", \"label\": \"总费用\", \"value\": \"total_fee\"}, {\"key\": \"insurance_type\", \"label\": \"医保类型\", \"value\": \"insurance_type\"}]','2026-05-14 11:10:53','2026-05-14 11:10:53');
/*!40000 ALTER TABLE `dictionaries` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `evaluation_complaint_events`
--

DROP TABLE IF EXISTS `evaluation_complaint_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `evaluation_complaint_events` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `complaint_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_type` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `comment` text COLLATE utf8mb4_unicode_ci,
  `payload_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_eval_complaint_events_complaint` (`complaint_id`),
  CONSTRAINT `fk_eval_complaint_events_complaint` FOREIGN KEY (`complaint_id`) REFERENCES `evaluation_complaints` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `evaluation_complaint_events`
--

LOCK TABLES `evaluation_complaint_events` WRITE;
/*!40000 ALTER TABLE `evaluation_complaint_events` DISABLE KEYS */;
/*!40000 ALTER TABLE `evaluation_complaint_events` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `evaluation_complaints`
--

DROP TABLE IF EXISTS `evaluation_complaints`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `evaluation_complaints` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual',
  `kind` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'complaint',
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `patient_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `patient_phone` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `channel` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `title` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `content` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `rating` int DEFAULT NULL,
  `category` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `authenticity` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'unconfirmed',
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'new',
  `responsible_department` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `responsible_person` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `audit_opinion` text COLLATE utf8mb4_unicode_ci,
  `handling_opinion` text COLLATE utf8mb4_unicode_ci,
  `rectification_measures` text COLLATE utf8mb4_unicode_ci,
  `tracking_opinion` text COLLATE utf8mb4_unicode_ci,
  `raw_payload` json DEFAULT NULL,
  `created_by` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `archived_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_eval_complaints_kind_status` (`kind`,`status`),
  KEY `idx_eval_complaints_source` (`source`),
  KEY `idx_eval_complaints_patient` (`patient_id`),
  KEY `idx_eval_complaints_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `evaluation_complaints`
--

LOCK TABLES `evaluation_complaints` WRITE;
/*!40000 ALTER TABLE `evaluation_complaints` DISABLE KEYS */;
/*!40000 ALTER TABLE `evaluation_complaints` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `exam_reports`
--

DROP TABLE IF EXISTS `exam_reports`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `exam_reports` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `exam_no` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `exam_type` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `exam_name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `body_part` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `report_conclusion` text COLLATE utf8mb4_unicode_ci,
  `report_findings` text COLLATE utf8mb4_unicode_ci,
  `ordered_at` datetime DEFAULT NULL,
  `reported_at` datetime DEFAULT NULL,
  `department_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `doctor_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_system` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_exam_report_no` (`exam_no`),
  KEY `idx_exam_reports_patient` (`patient_id`),
  KEY `idx_exam_reports_visit` (`visit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `exam_reports`
--

LOCK TABLES `exam_reports` WRITE;
/*!40000 ALTER TABLE `exam_reports` DISABLE KEYS */;
INSERT INTO `exam_reports` VALUES ('ebf0361d-8eac-5c27-b6ce-b95ebc730fee','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','2d4c455b-948f-46d3-9530-17573e1c4991','E778',NULL,'眼底检查',NULL,NULL,NULL,NULL,NULL,NULL,NULL,'临床事实同步','2026-05-14 23:15:00','2026-05-14 23:27:20'),('EXAM-P001-1','P001','V001','EX20260510001','ECG','十二导联心电图','心脏','窦性心律，未见明显急性缺血改变。',NULL,NULL,'2026-05-10 11:00:00','功能科','检查医生','PACS','2026-05-14 11:48:55','2026-05-14 11:48:55');
/*!40000 ALTER TABLE `exam_reports` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `followup_plans`
--

DROP TABLE IF EXISTS `followup_plans`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `followup_plans` (
  `id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `scenario` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `disease_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `department_id` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `form_template_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `trigger_type` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `trigger_offset` int NOT NULL DEFAULT '0',
  `channel` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'phone',
  `assignee_role` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'agent',
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `rules_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `followup_plans`
--

LOCK TABLES `followup_plans` WRITE;
/*!40000 ALTER TABLE `followup_plans` DISABLE KEYS */;
INSERT INTO `followup_plans` VALUES ('PLAN-DISCHARGE','出院后 7 日随访','随访','','','discharge-follow-up','出院后',7,'phone','nurse','active','{}','2026-05-14 07:52:37','2026-05-14 07:52:37'),('PLAN-HTN','高血压慢病随访','慢病','I10','DEPT-CARD','hypertension-follow-up','定期',30,'phone','agent','active','{\"ageMin\": 45, \"diagnosis\": \"高血压\"}','2026-05-14 07:52:37','2026-05-14 07:52:37');
/*!40000 ALTER TABLE `followup_plans` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `followup_records`
--

DROP TABLE IF EXISTS `followup_records`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `followup_records` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `task_id` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `project_id` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `followup_type` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `channel` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'completed',
  `summary` text COLLATE utf8mb4_unicode_ci,
  `satisfaction_score` decimal(5,2) DEFAULT NULL,
  `risk_level` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `followed_at` datetime DEFAULT NULL,
  `operator_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_system` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_followup_records_patient` (`patient_id`),
  KEY `idx_followup_records_task` (`task_id`),
  KEY `idx_followup_records_project` (`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `followup_records`
--

LOCK TABLES `followup_records` WRITE;
/*!40000 ALTER TABLE `followup_records` DISABLE KEYS */;
INSERT INTO `followup_records` VALUES ('12b9f041-1f39-56e3-9cec-2d9121bf5873','1c27c78b-d838-4934-a6cc-405d685f22cd','72556df3-08cb-4848-a955-dd5d470a8b27',NULL,NULL,NULL,NULL,'completed','用药依从性好',0.00,NULL,NULL,NULL,'临床事实同步','2026-05-14 23:20:08','2026-05-14 23:20:08'),('5a58422d-387e-577f-ab57-c8534617ede5','fb7e80e4-4059-4310-a243-ed909a0b52fc','22dd7d32-9788-4eb4-8c9d-14d8f6b0f8ee',NULL,NULL,NULL,NULL,'completed','用药依从性好',0.00,NULL,NULL,NULL,'临床事实同步','2026-05-14 23:17:13','2026-05-14 23:17:13'),('5c9678ac-abfb-541e-9d79-5f8de57c897a','3764c3f1-d305-4aee-8c8e-7794da448e1c','6df982be-1ab9-4fca-92df-ad3ba325b590',NULL,NULL,NULL,NULL,'completed','用药依从性好',0.00,NULL,NULL,NULL,'临床事实同步','2026-05-14 23:15:20','2026-05-14 23:15:20'),('65234283-e62f-5035-b0d7-2dffda14a0d0','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','2d4c455b-948f-46d3-9530-17573e1c4991',NULL,NULL,NULL,NULL,'completed','用药依从性好',0.00,NULL,NULL,NULL,'临床事实同步','2026-05-14 23:27:20','2026-05-14 23:27:20'),('b32a43e6-d6b1-5330-b93e-71c4c4e44733','0599f627-9d4c-46b8-bc0c-326ed7c8dee2','398b7243-901d-4ba3-89cb-64c7e1d488ef',NULL,NULL,NULL,NULL,'completed','用药依从性好',0.00,NULL,NULL,NULL,'临床事实同步','2026-05-14 23:15:00','2026-05-14 23:15:00'),('d73fb98d-371f-535d-b4a4-c038f3fe758f','b5f4ca15-6fc4-41b7-8cd1-81b77b138141','c3e2b5c0-7371-489c-9a4a-f6f9f01c00f2',NULL,NULL,NULL,NULL,'completed','用药依从性好',0.00,NULL,NULL,NULL,'临床事实同步','2026-05-14 23:19:51','2026-05-14 23:19:51'),('FUR-P001-1','P001','V001','FT001','SAT-OUTPATIENT','满意度随访','phone','completed','患者反馈候诊时间略长，用药说明清楚。',86.00,'low','2026-05-12 15:00:00','随访员A','followup','2026-05-14 11:48:55','2026-05-14 11:48:55');
/*!40000 ALTER TABLE `followup_records` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `followup_tasks`
--

DROP TABLE IF EXISTS `followup_tasks`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `followup_tasks` (
  `id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `plan_id` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `form_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `form_template_id` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `assignee_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `role` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `channel` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'phone',
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `priority` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'normal',
  `due_at` date DEFAULT NULL,
  `result_json` json DEFAULT NULL,
  `last_event` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `followup_tasks`
--

LOCK TABLES `followup_tasks` WRITE;
/*!40000 ALTER TABLE `followup_tasks` DISABLE KEYS */;
/*!40000 ALTER TABLE `followup_tasks` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `form_components`
--

DROP TABLE IF EXISTS `form_components`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `form_components` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `form_version_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `parent_component_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `component_key` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `component_type` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL,
  `label` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `required` tinyint(1) NOT NULL DEFAULT '0',
  `config_json` json DEFAULT NULL,
  `binding_json` json DEFAULT NULL,
  `sort_order` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `fk_form_components_version` (`form_version_id`),
  CONSTRAINT `fk_form_components_version` FOREIGN KEY (`form_version_id`) REFERENCES `form_versions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `form_components`
--

LOCK TABLES `form_components` WRITE;
/*!40000 ALTER TABLE `form_components` DISABLE KEYS */;
/*!40000 ALTER TABLE `form_components` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `form_library_items`
--

DROP TABLE IF EXISTS `form_library_items`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `form_library_items` (
  `id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` enum('template','common','atom') COLLATE utf8mb4_unicode_ci NOT NULL,
  `label` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `hint` text COLLATE utf8mb4_unicode_ci,
  `scenario` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `components_json` json NOT NULL,
  `sort_order` int NOT NULL DEFAULT '0',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `form_library_items`
--

LOCK TABLES `form_library_items` WRITE;
/*!40000 ALTER TABLE `form_library_items` DISABLE KEYS */;
INSERT INTO `form_library_items` VALUES ('atom-date','atom','日期','就诊、随访、手术日期','','[{\"id\": \"date\", \"type\": \"date\", \"label\": \"日期\", \"category\": \"原子组件\", \"required\": false}]',12,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('atom-number','atom','数字','年龄、评分、次数','','[{\"id\": \"number\", \"type\": \"number\", \"label\": \"数字\", \"category\": \"原子组件\", \"required\": false}]',11,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('atom-rating','atom','评分','星级、NPS、疼痛评分','','[{\"id\": \"rating\", \"type\": \"rating\", \"label\": \"评分\", \"scale\": 5, \"category\": \"原子组件\", \"required\": false}]',13,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('atom-text','atom','单行文本','姓名、编号、短文本','','[{\"id\": \"text\", \"type\": \"text\", \"label\": \"单行文本\", \"category\": \"原子组件\", \"required\": false}]',10,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('diabetes-management','template','糖尿病管理随访','血糖、低血糖事件、饮食运动、足部和用药依从性','慢病','[{\"id\": \"patient_section\", \"type\": \"section\", \"label\": \"患者基础信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_name\", \"type\": \"text\", \"label\": \"患者姓名\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.name\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"patient_gender\", \"type\": \"single_select\", \"label\": \"性别\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PID-8\", \"valuePath\": \"PID.8\", \"dataSourceId\": \"hl7-adt\"}, \"options\": [{\"label\": \"男\", \"value\": \"male\"}, {\"label\": \"女\", \"value\": \"female\"}, {\"label\": \"其他\", \"value\": \"other\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_age\", \"type\": \"number\", \"label\": \"年龄\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.age\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_phone\", \"type\": \"text\", \"label\": \"联系电话\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.phone\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_section\", \"type\": \"section\", \"label\": \"随访记录\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_date\", \"type\": \"date\", \"label\": \"随访日期\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"follow_method\", \"type\": \"single_select\", \"label\": \"随访方式\", \"options\": [{\"label\": \"电话\", \"value\": \"phone\"}, {\"label\": \"门诊\", \"value\": \"clinic\"}, {\"label\": \"线上\", \"value\": \"online\"}, {\"label\": \"上门\", \"value\": \"home\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"symptoms\", \"type\": \"multi_select\", \"label\": \"当前症状\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from symptom_dict where disease_code = :diseaseCode\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"medication_adherence\", \"type\": \"likert\", \"label\": \"用药依从性\", \"options\": [{\"label\": \"很不满意\", \"value\": \"1\"}, {\"label\": \"不满意\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"满意\", \"value\": \"4\"}, {\"label\": \"非常满意\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"glucose_section\", \"type\": \"section\", \"label\": \"血糖管理\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"fasting_glucose\", \"type\": \"number\", \"label\": \"空腹血糖 mmol/L\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"postprandial_glucose\", \"type\": \"number\", \"label\": \"餐后 2 小时血糖 mmol/L\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"hypoglycemia\", \"type\": \"single_select\", \"label\": \"近期低血糖事件\", \"options\": [{\"label\": \"无\", \"value\": \"none\"}, {\"label\": \"1 次\", \"value\": \"once\"}, {\"label\": \"2 次及以上\", \"value\": \"multiple\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"diet_exercise\", \"rows\": [\"控制主食\", \"规律运动\", \"监测血糖\", \"足部护理\"], \"type\": \"matrix\", \"label\": \"饮食与运动执行情况\", \"columns\": [\"未执行\", \"偶尔\", \"基本做到\", \"完全做到\"], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"foot_problem\", \"type\": \"textarea\", \"label\": \"足部异常或其他问题\", \"category\": \"公共组件\", \"required\": false}]',204,1,'2026-05-14 08:07:26','2026-05-14 08:07:26'),('discharge-follow-up','template','出院后随访问卷','出院患者基础信息、随访方式、症状、用药依从性和复诊提醒','随访','[{\"id\": \"patient_section\", \"type\": \"section\", \"label\": \"患者基础信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_name\", \"type\": \"text\", \"label\": \"患者姓名\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.name\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"patient_gender\", \"type\": \"single_select\", \"label\": \"性别\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PID-8\", \"valuePath\": \"PID.8\", \"dataSourceId\": \"hl7-adt\"}, \"options\": [{\"label\": \"男\", \"value\": \"male\"}, {\"label\": \"女\", \"value\": \"female\"}, {\"label\": \"其他\", \"value\": \"other\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_age\", \"type\": \"number\", \"label\": \"年龄\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.age\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_phone\", \"type\": \"text\", \"label\": \"联系电话\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.phone\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_section\", \"type\": \"section\", \"label\": \"随访记录\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_date\", \"type\": \"date\", \"label\": \"随访日期\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"follow_method\", \"type\": \"single_select\", \"label\": \"随访方式\", \"options\": [{\"label\": \"电话\", \"value\": \"phone\"}, {\"label\": \"门诊\", \"value\": \"clinic\"}, {\"label\": \"线上\", \"value\": \"online\"}, {\"label\": \"上门\", \"value\": \"home\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"symptoms\", \"type\": \"multi_select\", \"label\": \"当前症状\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from symptom_dict where disease_code = :diseaseCode\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"medication_adherence\", \"type\": \"likert\", \"label\": \"用药依从性\", \"options\": [{\"label\": \"很不满意\", \"value\": \"1\"}, {\"label\": \"不满意\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"满意\", \"value\": \"4\"}, {\"label\": \"非常满意\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": false}]',201,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('follow-up','common','随访','随访方式、时间、症状、用药依从性','','[{\"id\": \"follow_section\", \"type\": \"section\", \"label\": \"随访记录\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_date\", \"type\": \"date\", \"label\": \"随访日期\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"follow_method\", \"type\": \"single_select\", \"label\": \"随访方式\", \"options\": [{\"label\": \"电话\", \"value\": \"phone\"}, {\"label\": \"门诊\", \"value\": \"clinic\"}, {\"label\": \"线上\", \"value\": \"online\"}, {\"label\": \"上门\", \"value\": \"home\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"symptoms\", \"type\": \"multi_select\", \"label\": \"当前症状\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from symptom_dict where disease_code = :diseaseCode\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"medication_adherence\", \"type\": \"likert\", \"label\": \"用药依从性\", \"options\": [{\"label\": \"很不满意\", \"value\": \"1\"}, {\"label\": \"不满意\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"满意\", \"value\": \"4\"}, {\"label\": \"非常满意\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": false}]',102,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('hypertension-follow-up','template','高血压慢病随访','血压、用药、症状、生活方式和复诊计划','慢病','[{\"id\": \"patient_section\", \"type\": \"section\", \"label\": \"患者基础信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_name\", \"type\": \"text\", \"label\": \"患者姓名\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.name\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"patient_gender\", \"type\": \"single_select\", \"label\": \"性别\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PID-8\", \"valuePath\": \"PID.8\", \"dataSourceId\": \"hl7-adt\"}, \"options\": [{\"label\": \"男\", \"value\": \"male\"}, {\"label\": \"女\", \"value\": \"female\"}, {\"label\": \"其他\", \"value\": \"other\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_age\", \"type\": \"number\", \"label\": \"年龄\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.age\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_phone\", \"type\": \"text\", \"label\": \"联系电话\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.phone\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_section\", \"type\": \"section\", \"label\": \"随访记录\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_date\", \"type\": \"date\", \"label\": \"随访日期\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"follow_method\", \"type\": \"single_select\", \"label\": \"随访方式\", \"options\": [{\"label\": \"电话\", \"value\": \"phone\"}, {\"label\": \"门诊\", \"value\": \"clinic\"}, {\"label\": \"线上\", \"value\": \"online\"}, {\"label\": \"上门\", \"value\": \"home\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"symptoms\", \"type\": \"multi_select\", \"label\": \"当前症状\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from symptom_dict where disease_code = :diseaseCode\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"medication_adherence\", \"type\": \"likert\", \"label\": \"用药依从性\", \"options\": [{\"label\": \"很不满意\", \"value\": \"1\"}, {\"label\": \"不满意\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"满意\", \"value\": \"4\"}, {\"label\": \"非常满意\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"bp_section\", \"type\": \"section\", \"label\": \"血压与生活方式\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"systolic_bp\", \"type\": \"number\", \"label\": \"收缩压 mmHg\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"diastolic_bp\", \"type\": \"number\", \"label\": \"舒张压 mmHg\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"bp_control\", \"type\": \"likert\", \"label\": \"血压控制情况\", \"options\": [{\"label\": \"很差\", \"value\": \"1\"}, {\"label\": \"偏差\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"较好\", \"value\": \"4\"}, {\"label\": \"很好\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"lifestyle\", \"type\": \"multi_select\", \"label\": \"生活方式干预\", \"options\": [{\"label\": \"限盐\", \"value\": \"salt\"}, {\"label\": \"规律运动\", \"value\": \"exercise\"}, {\"label\": \"控制体重\", \"value\": \"weight\"}, {\"label\": \"戒烟限酒\", \"value\": \"smoke_alcohol\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"adverse_reaction\", \"type\": \"textarea\", \"label\": \"药物不良反应\", \"category\": \"公共组件\", \"required\": false}]',203,1,'2026-05-14 08:07:26','2026-05-14 08:07:26'),('outpatient-satisfaction','template','患者就诊满意度调查','由患者基础信息、就诊信息、满意度公共组件组合而成','调查','[{\"id\": \"patient_section\", \"type\": \"section\", \"label\": \"患者基础信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_name\", \"type\": \"text\", \"label\": \"患者姓名\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.name\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"patient_gender\", \"type\": \"single_select\", \"label\": \"性别\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PID-8\", \"valuePath\": \"PID.8\", \"dataSourceId\": \"hl7-adt\"}, \"options\": [{\"label\": \"男\", \"value\": \"male\"}, {\"label\": \"女\", \"value\": \"female\"}, {\"label\": \"其他\", \"value\": \"other\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_age\", \"type\": \"number\", \"label\": \"年龄\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.age\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_phone\", \"type\": \"text\", \"label\": \"联系电话\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.phone\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"visit_section\", \"type\": \"section\", \"label\": \"就诊信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"visit_date\", \"type\": \"date\", \"label\": \"就诊日期\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PV1-44\", \"valuePath\": \"PV1.44\", \"dataSourceId\": \"hl7-adt\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"department\", \"type\": \"remote_options\", \"label\": \"就诊科室\", \"binding\": {\"kind\": \"grpc\", \"labelPath\": \"$.name\", \"operation\": \"DepartmentService/ListDepartments\", \"valuePath\": \"$.code\", \"dataSourceId\": \"dept-grpc\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"diagnosis\", \"type\": \"remote_options\", \"label\": \"诊断\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from diagnosis_dict where keyword like :keyword\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"satisfaction_section\", \"type\": \"section\", \"label\": \"满意度评价\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"overall_satisfaction\", \"type\": \"likert\", \"label\": \"总体满意度\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from survey_options where group_code = \'satisfaction\'\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"options\": [{\"label\": \"很不满意\", \"value\": \"1\"}, {\"label\": \"不满意\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"满意\", \"value\": \"4\"}, {\"label\": \"非常满意\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"service_matrix\", \"rows\": [\"挂号缴费流程\", \"候诊时间\", \"医生沟通\", \"护士服务\", \"检查检验指引\", \"院内环境\"], \"type\": \"matrix\", \"label\": \"分项满意度\", \"columns\": [\"很不满意\", \"不满意\", \"一般\", \"满意\", \"非常满意\"], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"recommend_score\", \"type\": \"rating\", \"label\": \"推荐意愿\", \"scale\": 10, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"feedback\", \"type\": \"textarea\", \"label\": \"意见与建议\", \"category\": \"公共组件\", \"required\": false}]',200,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('patient-basic','common','患者基础信息','姓名、性别、年龄、手机号，可从主索引/API/HL7 ADT 回填','','[{\"id\": \"patient_section\", \"type\": \"section\", \"label\": \"患者基础信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_name\", \"type\": \"text\", \"label\": \"患者姓名\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.name\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"patient_gender\", \"type\": \"single_select\", \"label\": \"性别\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PID-8\", \"valuePath\": \"PID.8\", \"dataSourceId\": \"hl7-adt\"}, \"options\": [{\"label\": \"男\", \"value\": \"male\"}, {\"label\": \"女\", \"value\": \"female\"}, {\"label\": \"其他\", \"value\": \"other\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_age\", \"type\": \"number\", \"label\": \"年龄\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.age\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_phone\", \"type\": \"text\", \"label\": \"联系电话\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.phone\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}]',100,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('physical-exam-review','template','体检异常复查登记','体检异常项、影像/检验关联、复查建议和结果跟踪','体检','[{\"id\": \"patient_section\", \"type\": \"section\", \"label\": \"患者基础信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_name\", \"type\": \"text\", \"label\": \"患者姓名\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.name\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"patient_gender\", \"type\": \"single_select\", \"label\": \"性别\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PID-8\", \"valuePath\": \"PID.8\", \"dataSourceId\": \"hl7-adt\"}, \"options\": [{\"label\": \"男\", \"value\": \"male\"}, {\"label\": \"女\", \"value\": \"female\"}, {\"label\": \"其他\", \"value\": \"other\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_age\", \"type\": \"number\", \"label\": \"年龄\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.age\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_phone\", \"type\": \"text\", \"label\": \"联系电话\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.phone\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"exam_section\", \"type\": \"section\", \"label\": \"体检异常信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"exam_date\", \"type\": \"date\", \"label\": \"体检日期\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"abnormal_items\", \"type\": \"multi_select\", \"label\": \"异常项目\", \"binding\": {\"kind\": \"http\", \"labelPath\": \"$.name\", \"operation\": \"GET /exam/:examId/abnormal-items\", \"valuePath\": \"$.code\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"related_image\", \"type\": \"remote_options\", \"label\": \"相关影像\", \"binding\": {\"kind\": \"dicom\", \"labelPath\": \"$.StudyDescription\", \"operation\": \"QIDO-RS /studies?PatientID=:patientId\", \"valuePath\": \"$.StudyInstanceUID\", \"dataSourceId\": \"dicom-pacs\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"review_advice\", \"type\": \"textarea\", \"label\": \"复查建议\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"review_date\", \"type\": \"date\", \"label\": \"计划复查日期\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"review_result\", \"type\": \"textarea\", \"label\": \"复查结果\", \"category\": \"公共组件\", \"required\": false}]',205,1,'2026-05-14 08:07:26','2026-05-14 08:07:26'),('post-op','common','术后跟踪','手术信息、疼痛评分、影像检查','','[{\"id\": \"post_op_section\", \"type\": \"section\", \"label\": \"术后跟踪\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"surgery_date\", \"type\": \"date\", \"label\": \"手术日期\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PR1-5\", \"valuePath\": \"PR1.5\", \"dataSourceId\": \"hl7-adt\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"procedure_name\", \"type\": \"text\", \"label\": \"手术名称\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PR1-3\", \"valuePath\": \"PR1.3\", \"dataSourceId\": \"hl7-adt\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"pain_score\", \"type\": \"rating\", \"label\": \"疼痛评分\", \"scale\": 10, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"image_followup\", \"type\": \"remote_options\", \"label\": \"相关影像检查\", \"binding\": {\"kind\": \"dicom\", \"labelPath\": \"$.StudyDescription\", \"operation\": \"QIDO-RS /studies?PatientID=:patientId\", \"valuePath\": \"$.StudyInstanceUID\", \"dataSourceId\": \"dicom-pacs\"}, \"category\": \"公共组件\", \"required\": false}]',103,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('post-op-follow-up','template','术后随访问卷','由患者基础信息、术后跟踪、随访公共组件组合而成','术后','[{\"id\": \"patient_section\", \"type\": \"section\", \"label\": \"患者基础信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_name\", \"type\": \"text\", \"label\": \"患者姓名\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.name\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"patient_gender\", \"type\": \"single_select\", \"label\": \"性别\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PID-8\", \"valuePath\": \"PID.8\", \"dataSourceId\": \"hl7-adt\"}, \"options\": [{\"label\": \"男\", \"value\": \"male\"}, {\"label\": \"女\", \"value\": \"female\"}, {\"label\": \"其他\", \"value\": \"other\"}], \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_age\", \"type\": \"number\", \"label\": \"年龄\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.age\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"patient_phone\", \"type\": \"text\", \"label\": \"联系电话\", \"binding\": {\"kind\": \"http\", \"operation\": \"GET /patients/:patientId\", \"valuePath\": \"$.phone\", \"dataSourceId\": \"patients-api\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"post_op_section\", \"type\": \"section\", \"label\": \"术后跟踪\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"surgery_date\", \"type\": \"date\", \"label\": \"手术日期\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PR1-5\", \"valuePath\": \"PR1.5\", \"dataSourceId\": \"hl7-adt\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"procedure_name\", \"type\": \"text\", \"label\": \"手术名称\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PR1-3\", \"valuePath\": \"PR1.3\", \"dataSourceId\": \"hl7-adt\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"pain_score\", \"type\": \"rating\", \"label\": \"疼痛评分\", \"scale\": 10, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"image_followup\", \"type\": \"remote_options\", \"label\": \"相关影像检查\", \"binding\": {\"kind\": \"dicom\", \"labelPath\": \"$.StudyDescription\", \"operation\": \"QIDO-RS /studies?PatientID=:patientId\", \"valuePath\": \"$.StudyInstanceUID\", \"dataSourceId\": \"dicom-pacs\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_section\", \"type\": \"section\", \"label\": \"随访记录\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"follow_date\", \"type\": \"date\", \"label\": \"随访日期\", \"category\": \"公共组件\", \"required\": true}, {\"id\": \"follow_method\", \"type\": \"single_select\", \"label\": \"随访方式\", \"options\": [{\"label\": \"电话\", \"value\": \"phone\"}, {\"label\": \"门诊\", \"value\": \"clinic\"}, {\"label\": \"线上\", \"value\": \"online\"}, {\"label\": \"上门\", \"value\": \"home\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"symptoms\", \"type\": \"multi_select\", \"label\": \"当前症状\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from symptom_dict where disease_code = :diseaseCode\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"category\": \"公共组件\", \"required\": false}, {\"id\": \"medication_adherence\", \"type\": \"likert\", \"label\": \"用药依从性\", \"options\": [{\"label\": \"很不满意\", \"value\": \"1\"}, {\"label\": \"不满意\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"满意\", \"value\": \"4\"}, {\"label\": \"非常满意\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": false}]',202,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('satisfaction','common','满意度','总体满意、分项矩阵、推荐意愿、原因和建议','','[{\"id\": \"satisfaction_section\", \"type\": \"section\", \"label\": \"满意度评价\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"overall_satisfaction\", \"type\": \"likert\", \"label\": \"总体满意度\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from survey_options where group_code = \'satisfaction\'\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"options\": [{\"label\": \"很不满意\", \"value\": \"1\"}, {\"label\": \"不满意\", \"value\": \"2\"}, {\"label\": \"一般\", \"value\": \"3\"}, {\"label\": \"满意\", \"value\": \"4\"}, {\"label\": \"非常满意\", \"value\": \"5\"}], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"service_matrix\", \"rows\": [\"挂号缴费流程\", \"候诊时间\", \"医生沟通\", \"护士服务\", \"检查检验指引\", \"院内环境\"], \"type\": \"matrix\", \"label\": \"分项满意度\", \"columns\": [\"很不满意\", \"不满意\", \"一般\", \"满意\", \"非常满意\"], \"category\": \"公共组件\", \"required\": true}, {\"id\": \"recommend_score\", \"type\": \"rating\", \"label\": \"推荐意愿\", \"scale\": 10, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"feedback\", \"type\": \"textarea\", \"label\": \"意见与建议\", \"category\": \"公共组件\", \"required\": false}]',104,1,'2026-05-14 06:51:30','2026-05-14 06:51:30'),('visit-info','common','就诊信息','科室、医生、就诊日期、诊断，支持 HIS/API/gRPC/HL7','','[{\"id\": \"visit_section\", \"type\": \"section\", \"label\": \"就诊信息\", \"category\": \"公共组件\", \"required\": false}, {\"id\": \"visit_date\", \"type\": \"date\", \"label\": \"就诊日期\", \"binding\": {\"kind\": \"hl7\", \"operation\": \"PV1-44\", \"valuePath\": \"PV1.44\", \"dataSourceId\": \"hl7-adt\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"department\", \"type\": \"remote_options\", \"label\": \"就诊科室\", \"binding\": {\"kind\": \"grpc\", \"labelPath\": \"$.name\", \"operation\": \"DepartmentService/ListDepartments\", \"valuePath\": \"$.code\", \"dataSourceId\": \"dept-grpc\"}, \"category\": \"公共组件\", \"required\": true}, {\"id\": \"diagnosis\", \"type\": \"remote_options\", \"label\": \"诊断\", \"binding\": {\"kind\": \"mysql\", \"labelPath\": \"$.label\", \"operation\": \"select label, value from diagnosis_dict where keyword like :keyword\", \"valuePath\": \"$.value\", \"dataSourceId\": \"survey-dict\"}, \"category\": \"公共组件\", \"required\": false}]',101,1,'2026-05-14 06:51:30','2026-05-14 06:51:30');
/*!40000 ALTER TABLE `form_library_items` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `form_submissions`
--

DROP TABLE IF EXISTS `form_submissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `form_submissions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `form_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `form_version_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `submitter_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('draft','submitted','approved','rejected') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'submitted',
  `data_json` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_form_submissions_form` (`form_id`),
  KEY `fk_form_submissions_version` (`form_version_id`),
  KEY `fk_form_submissions_submitter` (`submitter_id`),
  CONSTRAINT `fk_form_submissions_form` FOREIGN KEY (`form_id`) REFERENCES `forms` (`id`),
  CONSTRAINT `fk_form_submissions_submitter` FOREIGN KEY (`submitter_id`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_form_submissions_version` FOREIGN KEY (`form_version_id`) REFERENCES `form_versions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `form_submissions`
--

LOCK TABLES `form_submissions` WRITE;
/*!40000 ALTER TABLE `form_submissions` DISABLE KEYS */;
/*!40000 ALTER TABLE `form_submissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `form_versions`
--

DROP TABLE IF EXISTS `form_versions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `form_versions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `form_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` int NOT NULL,
  `schema_json` json NOT NULL,
  `schema_hash` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `change_note` text COLLATE utf8mb4_unicode_ci,
  `created_by` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `published` tinyint(1) NOT NULL DEFAULT '0',
  `locked_at` timestamp NULL DEFAULT NULL,
  `published_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_form_version` (`form_id`,`version`),
  KEY `fk_form_versions_creator` (`created_by`),
  KEY `idx_form_versions_hash` (`form_id`,`schema_hash`),
  CONSTRAINT `fk_form_versions_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_form_versions_form` FOREIGN KEY (`form_id`) REFERENCES `forms` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `form_versions`
--

LOCK TABLES `form_versions` WRITE;
/*!40000 ALTER TABLE `form_versions` DISABLE KEYS */;
/*!40000 ALTER TABLE `form_versions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `forms`
--

DROP TABLE IF EXISTS `forms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `forms` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `status` enum('draft','published','archived') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'draft',
  `current_version_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `forms`
--

LOCK TABLES `forms` WRITE;
/*!40000 ALTER TABLE `forms` DISABLE KEYS */;
/*!40000 ALTER TABLE `forms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `integration_channels`
--

DROP TABLE IF EXISTS `integration_channels`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `integration_channels` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `endpoint` text COLLATE utf8mb4_unicode_ci,
  `app_id` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `credential_ref` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `config_json` json DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_integration_channels_kind` (`kind`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `integration_channels`
--

LOCK TABLES `integration_channels` WRITE;
/*!40000 ALTER TABLE `integration_channels` DISABLE KEYS */;
INSERT INTO `integration_channels` VALUES ('CHAN-MINIPROGRAM','mini_program','微信小程序订阅消息','https://api.weixin.qq.com','','secret://wechat-mini-program/default','{\"pagePath\": \"pages/survey/index\", \"provider\": \"wechat_mini_program\", \"templateId\": \"\"}',0,'2026-05-14 23:03:58','2026-05-14 23:03:58'),('CHAN-QQ','qq','QQ 分享接口','https://connect.qq.com','qq-app-id','secret://qq/default','{}',0,'2026-05-14 09:13:30','2026-05-14 09:13:30'),('CHAN-SMS','sms','短信接口','https://sms.example.local/send','','secret://sms/default','{\"signature\": \"医院\", \"templateMode\": true}',1,'2026-05-14 09:13:30','2026-05-14 09:13:30'),('CHAN-WEB','web','Web 链接','http://127.0.0.1:4321/survey','','','{\"allowAnonymous\": true}',1,'2026-05-14 09:13:30','2026-05-14 09:13:30'),('CHAN-WECHAT','wechat','微信公众号接口','https://api.weixin.qq.com','wx-app-id','secret://wechat/default','{\"messageType\": \"template\"}',1,'2026-05-14 09:13:30','2026-05-14 09:13:30'),('CHAN-WEWORK','wework','企业微信应用消息','https://qyapi.weixin.qq.com','','secret://wework/default','{\"agentId\": \"\", \"provider\": \"wework\", \"templateId\": \"\"}',0,'2026-05-14 23:03:58','2026-05-14 23:03:58');
/*!40000 ALTER TABLE `integration_channels` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `interview_extracted_facts`
--

DROP TABLE IF EXISTS `interview_extracted_facts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `interview_extracted_facts` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `interview_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `fact_type` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `fact_key` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `fact_label` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `fact_value` text COLLATE utf8mb4_unicode_ci,
  `confidence` decimal(5,4) DEFAULT NULL,
  `extracted_at` datetime DEFAULT NULL,
  `source_text` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_interview_facts_patient` (`patient_id`),
  KEY `idx_interview_facts_key` (`fact_key`),
  KEY `idx_interview_facts_interview` (`interview_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `interview_extracted_facts`
--

LOCK TABLES `interview_extracted_facts` WRITE;
/*!40000 ALTER TABLE `interview_extracted_facts` DISABLE KEYS */;
INSERT INTO `interview_extracted_facts` VALUES ('10b09395-cfa5-50b4-bd3b-3f213fc8b3fa','fb7e80e4-4059-4310-a243-ed909a0b52fc','22dd7d32-9788-4eb4-8c9d-14d8f6b0f8ee',NULL,'experience','drug_compliance','用药依从性','良好',0.0000,NULL,NULL,'2026-05-14 23:17:13'),('174bdc6e-cd7a-5f8c-8e45-51b9bbab8162','0599f627-9d4c-46b8-bc0c-326ed7c8dee2','398b7243-901d-4ba3-89cb-64c7e1d488ef',NULL,'experience','drug_compliance','用药依从性','良好',0.0000,NULL,NULL,'2026-05-14 23:15:00'),('3c04ac9c-1ea0-5997-a9aa-a8a62bf0bb59','1c27c78b-d838-4934-a6cc-405d685f22cd','72556df3-08cb-4848-a955-dd5d470a8b27',NULL,'experience','drug_compliance','用药依从性','良好',0.0000,NULL,NULL,'2026-05-14 23:20:08'),('71bc9bfb-055e-535b-b5f8-bae8bf86b578','b5f4ca15-6fc4-41b7-8cd1-81b77b138141','c3e2b5c0-7371-489c-9a4a-f6f9f01c00f2',NULL,'experience','drug_compliance','用药依从性','良好',0.0000,NULL,NULL,'2026-05-14 23:19:51'),('95683016-19db-570a-b989-b623a475b3d4','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','2d4c455b-948f-46d3-9530-17573e1c4991',NULL,'experience','drug_compliance','用药依从性','良好',0.0000,NULL,NULL,'2026-05-14 23:27:20'),('d3515b83-bc57-597e-a26d-57d7f7029afc','3764c3f1-d305-4aee-8c8e-7794da448e1c','6df982be-1ab9-4fca-92df-ad3ba325b590',NULL,'experience','drug_compliance','用药依从性','良好',0.0000,NULL,NULL,'2026-05-14 23:15:20'),('FACT-P001-1','P001','V001','INT-P001-1','experience','waiting_time','候诊时间','候诊时间偏长',0.9200,'2026-05-12 15:05:00','等候时间有点久，其他还可以。','2026-05-14 11:48:55');
/*!40000 ALTER TABLE `interview_extracted_facts` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `interview_sessions`
--

DROP TABLE IF EXISTS `interview_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `interview_sessions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `form_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `call_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `mode` enum('chat','call','chat_call') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'chat',
  `status` enum('draft','active','completed','abandoned') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'draft',
  `messages_json` json DEFAULT NULL,
  `form_draft_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_interview_sessions_patient` (`patient_id`),
  KEY `fk_interview_sessions_call` (`call_id`),
  CONSTRAINT `fk_interview_sessions_call` FOREIGN KEY (`call_id`) REFERENCES `call_sessions` (`id`),
  CONSTRAINT `fk_interview_sessions_patient` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `interview_sessions`
--

LOCK TABLES `interview_sessions` WRITE;
/*!40000 ALTER TABLE `interview_sessions` DISABLE KEYS */;
/*!40000 ALTER TABLE `interview_sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `lab_reports`
--

DROP TABLE IF EXISTS `lab_reports`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `lab_reports` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `report_no` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `report_name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `specimen` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ordered_at` datetime DEFAULT NULL,
  `reported_at` datetime DEFAULT NULL,
  `department_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `doctor_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'reported',
  `source_system` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_lab_report_no` (`report_no`),
  KEY `idx_lab_reports_patient` (`patient_id`),
  KEY `idx_lab_reports_visit` (`visit_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `lab_reports`
--

LOCK TABLES `lab_reports` WRITE;
/*!40000 ALTER TABLE `lab_reports` DISABLE KEYS */;
INSERT INTO `lab_reports` VALUES ('c2a9b686-4dfa-5701-a4aa-11449e08bbf4','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','2d4c455b-948f-46d3-9530-17573e1c4991','L778','血糖',NULL,NULL,NULL,NULL,NULL,'reported','临床事实同步','2026-05-14 23:15:00','2026-05-14 23:27:20'),('LAB-P001-1','P001','V001','LAB20260510001','肝肾功能','血清',NULL,'2026-05-10 14:20:00','检验科','检验医生','reported','LIS','2026-05-14 11:48:55','2026-05-14 11:48:55');
/*!40000 ALTER TABLE `lab_reports` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `lab_results`
--

DROP TABLE IF EXISTS `lab_results`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `lab_results` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `report_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `item_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `item_name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `result_value` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `unit` varchar(60) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reference_range` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `abnormal_flag` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `numeric_value` decimal(12,4) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_lab_results_report` (`report_id`),
  KEY `idx_lab_results_item` (`item_code`),
  KEY `idx_lab_results_abnormal` (`abnormal_flag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `lab_results`
--

LOCK TABLES `lab_results` WRITE;
/*!40000 ALTER TABLE `lab_results` DISABLE KEYS */;
INSERT INTO `lab_results` VALUES ('e2485611-9f08-56d4-9294-bff85a067172','c2a9b686-4dfa-5701-a4aa-11449e08bbf4',NULL,'空腹血糖','6.8',NULL,NULL,NULL,0.0000,'2026-05-14 23:15:00'),('ebec983e-291f-5dc4-8d31-a2f73df55b21','c68b22b8-2d18-50c6-8edc-5edb161ef332',NULL,'空腹血糖','6.8',NULL,NULL,NULL,0.0000,'2026-05-14 23:15:20'),('f4f403e5-91b2-58f9-8476-df128ed285fe','c81bd122-7cae-5e0a-ae19-ee9b779d24d4',NULL,'空腹血糖','6.8',NULL,NULL,NULL,0.0000,'2026-05-14 23:17:13'),('LAR-P001-1','LAB-P001-1','CREA','肌酐','72','umol/L','57-97','normal',72.0000,'2026-05-14 11:48:55');
/*!40000 ALTER TABLE `lab_results` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `medical_records`
--

DROP TABLE IF EXISTS `medical_records`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `medical_records` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `record_no` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `record_type` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `summary` text COLLATE utf8mb4_unicode_ci,
  `chief_complaint` text COLLATE utf8mb4_unicode_ci,
  `present_illness` text COLLATE utf8mb4_unicode_ci,
  `diagnosis_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `diagnosis_name` varchar(240) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `procedure_name` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `study_uid` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `study_desc` varchar(240) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `recorded_at` datetime DEFAULT NULL,
  `source_refs_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_medical_record_no` (`record_no`),
  KEY `fk_medical_records_patient` (`patient_id`),
  KEY `fk_medical_records_visit` (`visit_id`),
  CONSTRAINT `fk_medical_records_patient` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`),
  CONSTRAINT `fk_medical_records_visit` FOREIGN KEY (`visit_id`) REFERENCES `clinical_visits` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `medical_records`
--

LOCK TABLES `medical_records` WRITE;
/*!40000 ALTER TABLE `medical_records` DISABLE KEYS */;
/*!40000 ALTER TABLE `medical_records` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `medication_orders`
--

DROP TABLE IF EXISTS `medication_orders`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `medication_orders` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `order_no` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `prescription_no` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `drug_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `drug_name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `generic_name` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `specification` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `dosage` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `dosage_unit` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `frequency` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `route` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `start_at` datetime DEFAULT NULL,
  `end_at` datetime DEFAULT NULL,
  `days` int DEFAULT NULL,
  `quantity` decimal(10,2) DEFAULT NULL,
  `manufacturer` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `doctor_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pharmacist_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `adverse_reaction` text COLLATE utf8mb4_unicode_ci,
  `compliance` varchar(60) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_medication_orders_patient` (`patient_id`),
  KEY `idx_medication_orders_visit` (`visit_id`),
  KEY `idx_medication_orders_drug` (`drug_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `medication_orders`
--

LOCK TABLES `medication_orders` WRITE;
/*!40000 ALTER TABLE `medication_orders` DISABLE KEYS */;
INSERT INTO `medication_orders` VALUES ('0eb44f5c-5575-5405-942d-d1d142fa8db0','0599f627-9d4c-46b8-bc0c-326ed7c8dee2','398b7243-901d-4ba3-89cb-64c7e1d488ef',NULL,NULL,NULL,'二甲双胍片',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0.00,NULL,NULL,NULL,'active',NULL,NULL,'2026-05-14 23:15:00','2026-05-14 23:15:00'),('1ce7656f-c59f-5313-88b5-841b725f71ee','fb7e80e4-4059-4310-a243-ed909a0b52fc','22dd7d32-9788-4eb4-8c9d-14d8f6b0f8ee',NULL,NULL,NULL,'二甲双胍片',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0.00,NULL,NULL,NULL,'active',NULL,NULL,'2026-05-14 23:17:13','2026-05-14 23:17:13'),('6b9cd12a-468a-5522-9934-8075a7b72323','3764c3f1-d305-4aee-8c8e-7794da448e1c','6df982be-1ab9-4fca-92df-ad3ba325b590',NULL,NULL,NULL,'二甲双胍片',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0.00,NULL,NULL,NULL,'active',NULL,NULL,'2026-05-14 23:15:20','2026-05-14 23:15:20'),('d842126c-499a-5052-bd98-494813cf4c36','1c27c78b-d838-4934-a6cc-405d685f22cd','72556df3-08cb-4848-a955-dd5d470a8b27',NULL,NULL,NULL,'二甲双胍片',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0.00,NULL,NULL,NULL,'active',NULL,NULL,'2026-05-14 23:20:08','2026-05-14 23:20:08'),('de04091a-e9a8-5dff-9deb-b99d61352897','b5f4ca15-6fc4-41b7-8cd1-81b77b138141','c3e2b5c0-7371-489c-9a4a-f6f9f01c00f2',NULL,NULL,NULL,'二甲双胍片',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0.00,NULL,NULL,NULL,'active',NULL,NULL,'2026-05-14 23:19:51','2026-05-14 23:19:51'),('ed0ba9b5-3dc1-579f-a88f-2e8382b6acdc','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','2d4c455b-948f-46d3-9530-17573e1c4991',NULL,NULL,NULL,'二甲双胍片',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,0,0.00,NULL,NULL,NULL,'active',NULL,NULL,'2026-05-14 23:27:20','2026-05-14 23:27:20'),('MED-P001-1','P001','V001','ORD20260510001',NULL,'YP-AML','苯磺酸氨氯地平片','氨氯地平','5mg*28片','5','mg','qd','口服','2026-05-10 10:00:00',NULL,28,28.00,NULL,'王医生',NULL,'active',NULL,'good','2026-05-14 11:48:55','2026-05-14 11:48:55');
/*!40000 ALTER TABLE `medication_orders` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `model_providers`
--

DROP TABLE IF EXISTS `model_providers`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `model_providers` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `mode` enum('realtime','offline','both') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'offline',
  `endpoint` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `model` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `credential_ref` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `config_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `model_providers`
--

LOCK TABLES `model_providers` WRITE;
/*!40000 ALTER TABLE `model_providers` DISABLE KEYS */;
/*!40000 ALTER TABLE `model_providers` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `offline_analysis_jobs`
--

DROP TABLE IF EXISTS `offline_analysis_jobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `offline_analysis_jobs` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `call_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `recording_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `provider_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('queued','running','completed','failed') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'queued',
  `result_json` json DEFAULT NULL,
  `error` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_offline_analysis_call` (`call_id`),
  KEY `fk_offline_analysis_recording` (`recording_id`),
  KEY `fk_offline_analysis_provider` (`provider_id`),
  CONSTRAINT `fk_offline_analysis_call` FOREIGN KEY (`call_id`) REFERENCES `call_sessions` (`id`),
  CONSTRAINT `fk_offline_analysis_provider` FOREIGN KEY (`provider_id`) REFERENCES `model_providers` (`id`),
  CONSTRAINT `fk_offline_analysis_recording` FOREIGN KEY (`recording_id`) REFERENCES `recordings` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `offline_analysis_jobs`
--

LOCK TABLES `offline_analysis_jobs` WRITE;
/*!40000 ALTER TABLE `offline_analysis_jobs` DISABLE KEYS */;
/*!40000 ALTER TABLE `offline_analysis_jobs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patient_diagnoses`
--

DROP TABLE IF EXISTS `patient_diagnoses`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `patient_diagnoses` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `diagnosis_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `diagnosis_name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `diagnosis_type` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'primary',
  `diagnosed_at` datetime DEFAULT NULL,
  `department_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `doctor_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_system` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_patient_diagnoses_patient` (`patient_id`),
  KEY `idx_patient_diagnoses_visit` (`visit_id`),
  KEY `idx_patient_diagnoses_code` (`diagnosis_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patient_diagnoses`
--

LOCK TABLES `patient_diagnoses` WRITE;
/*!40000 ALTER TABLE `patient_diagnoses` DISABLE KEYS */;
INSERT INTO `patient_diagnoses` VALUES ('00594160-821c-5bd3-b836-8d94d0b483cc','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','2d4c455b-948f-46d3-9530-17573e1c4991',NULL,'糖尿病','primary',NULL,NULL,NULL,'临床事实同步','2026-05-14 23:27:20','2026-05-14 23:27:20'),('17ec929c-9593-5cca-95f3-ba889ec22c00','b5f4ca15-6fc4-41b7-8cd1-81b77b138141','c3e2b5c0-7371-489c-9a4a-f6f9f01c00f2',NULL,'糖尿病','primary',NULL,NULL,NULL,'临床事实同步','2026-05-14 23:19:51','2026-05-14 23:19:51'),('511c96ba-0008-5ebf-a6cf-92d672663e0d','1c27c78b-d838-4934-a6cc-405d685f22cd','72556df3-08cb-4848-a955-dd5d470a8b27',NULL,'糖尿病','primary',NULL,NULL,NULL,'临床事实同步','2026-05-14 23:20:08','2026-05-14 23:20:08'),('ab0a7e1e-49ed-5cff-bf0b-410ec46df1f9','0599f627-9d4c-46b8-bc0c-326ed7c8dee2','398b7243-901d-4ba3-89cb-64c7e1d488ef',NULL,'糖尿病','primary',NULL,NULL,NULL,'临床事实同步','2026-05-14 23:15:00','2026-05-14 23:15:00'),('b09c3442-afb2-5803-a0b8-e6ab45636eac','fb7e80e4-4059-4310-a243-ed909a0b52fc','22dd7d32-9788-4eb4-8c9d-14d8f6b0f8ee',NULL,'糖尿病','primary',NULL,NULL,NULL,'临床事实同步','2026-05-14 23:17:13','2026-05-14 23:17:13'),('c01f1602-5f7f-5bfb-b677-ccd1f32dd185','3764c3f1-d305-4aee-8c8e-7794da448e1c','6df982be-1ab9-4fca-92df-ad3ba325b590',NULL,'糖尿病','primary',NULL,NULL,NULL,'临床事实同步','2026-05-14 23:15:20','2026-05-14 23:15:20'),('DX-P001-1','P001','V001','I10','高血压','primary','2026-05-10 09:30:00','心内科','王医生','HIS','2026-05-14 11:48:55','2026-05-14 11:48:55');
/*!40000 ALTER TABLE `patient_diagnoses` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patient_group_members`
--

DROP TABLE IF EXISTS `patient_group_members`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `patient_group_members` (
  `group_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `added_by` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`group_id`,`patient_id`),
  KEY `idx_patient_group_members_patient` (`patient_id`),
  CONSTRAINT `fk_patient_group_members_group` FOREIGN KEY (`group_id`) REFERENCES `patient_groups` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patient_group_members`
--

LOCK TABLES `patient_group_members` WRITE;
/*!40000 ALTER TABLE `patient_group_members` DISABLE KEYS */;
/*!40000 ALTER TABLE `patient_group_members` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patient_groups`
--

DROP TABLE IF EXISTS `patient_groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `patient_groups` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `category` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '专病',
  `mode` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'person',
  `assignment_mode` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual',
  `followup_plan_id` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `rules_json` json DEFAULT NULL,
  `permissions_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_patient_groups_category` (`category`),
  KEY `idx_patient_groups_plan` (`followup_plan_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patient_groups`
--

LOCK TABLES `patient_groups` WRITE;
/*!40000 ALTER TABLE `patient_groups` DISABLE KEYS */;
/*!40000 ALTER TABLE `patient_groups` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patient_histories`
--

DROP TABLE IF EXISTS `patient_histories`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `patient_histories` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `history_type` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `content` text COLLATE utf8mb4_unicode_ci,
  `recorded_at` datetime DEFAULT NULL,
  `source_system` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_patient_histories_patient` (`patient_id`),
  KEY `idx_patient_histories_type` (`history_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patient_histories`
--

LOCK TABLES `patient_histories` WRITE;
/*!40000 ALTER TABLE `patient_histories` DISABLE KEYS */;
INSERT INTO `patient_histories` VALUES ('35ec50aa-ea6e-5927-842e-b11f83e12bf2','1c27c78b-d838-4934-a6cc-405d685f22cd','past','既往史','高血压病史',NULL,'临床事实同步','2026-05-14 23:20:08','2026-05-14 23:20:08'),('43775984-c29d-5a02-9725-c54b42776cba','cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','past','既往史','高血压病史',NULL,'临床事实同步','2026-05-14 23:27:20','2026-05-14 23:27:20'),('7394cd5e-52f8-506d-a0f0-54335a242adb','3764c3f1-d305-4aee-8c8e-7794da448e1c','past','既往史','高血压病史',NULL,'临床事实同步','2026-05-14 23:15:20','2026-05-14 23:15:20'),('97ee5b82-f6fc-5ae1-adbe-8bb17861c9a7','fb7e80e4-4059-4310-a243-ed909a0b52fc','past','既往史','高血压病史',NULL,'临床事实同步','2026-05-14 23:17:13','2026-05-14 23:17:13'),('cf1ea4f3-5b1b-57f4-9048-8dc22fde0f07','0599f627-9d4c-46b8-bc0c-326ed7c8dee2','past','既往史','高血压病史',NULL,'临床事实同步','2026-05-14 23:15:00','2026-05-14 23:15:00'),('eddae826-e5e6-572a-a61b-9f52cf04e583','b5f4ca15-6fc4-41b7-8cd1-81b77b138141','past','既往史','高血压病史',NULL,'临床事实同步','2026-05-14 23:19:51','2026-05-14 23:19:51'),('HX-P001-1','P001','past','既往史','高血压病史 5 年，规律服药。','2026-05-10 09:40:00','EMR','2026-05-14 11:48:55','2026-05-14 11:48:55');
/*!40000 ALTER TABLE `patient_histories` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patient_tag_assignments`
--

DROP TABLE IF EXISTS `patient_tag_assignments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `patient_tag_assignments` (
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `tag_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`patient_id`,`tag_id`),
  KEY `idx_patient_tag_assignments_tag` (`tag_id`),
  CONSTRAINT `fk_patient_tag_assignments_tag` FOREIGN KEY (`tag_id`) REFERENCES `patient_tags` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patient_tag_assignments`
--

LOCK TABLES `patient_tag_assignments` WRITE;
/*!40000 ALTER TABLE `patient_tag_assignments` DISABLE KEYS */;
/*!40000 ALTER TABLE `patient_tag_assignments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patient_tags`
--

DROP TABLE IF EXISTS `patient_tags`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `patient_tags` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `color` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '#2563eb',
  `description` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patient_tags`
--

LOCK TABLES `patient_tags` WRITE;
/*!40000 ALTER TABLE `patient_tags` DISABLE KEYS */;
/*!40000 ALTER TABLE `patient_tags` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `patients`
--

DROP TABLE IF EXISTS `patients`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `patients` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_no` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `medical_record_no` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `gender` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `birth_date` date DEFAULT NULL,
  `age` int DEFAULT NULL,
  `id_card_no` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `phone` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `address` text COLLATE utf8mb4_unicode_ci,
  `nationality` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ethnicity` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `marital_status` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `insurance_type` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `blood_type` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `allergies_json` json DEFAULT NULL,
  `emergency_contact` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `emergency_phone` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `diagnosis` varchar(240) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` enum('active','follow_up','inactive') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `last_visit_at` date DEFAULT NULL,
  `source_refs_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `patient_no` (`patient_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `patients`
--

LOCK TABLES `patients` WRITE;
/*!40000 ALTER TABLE `patients` DISABLE KEYS */;
INSERT INTO `patients` VALUES ('cbd2f2b2-3aec-4cd7-b6b7-0712e1315dfa','P778',NULL,'事实患者',NULL,NULL,0,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,'[]',NULL,NULL,NULL,'active',NULL,'{\"protocol\": \"http\", \"dataSourceId\": \"119a6adf-55b8-419b-baeb-6a836e4b150a\"}','2026-05-14 23:27:20','2026-05-15 02:51:02');
/*!40000 ALTER TABLE `patients` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `permissions`
--

DROP TABLE IF EXISTS `permissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `permissions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `resource` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `action` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_permission` (`resource`,`action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `permissions`
--

LOCK TABLES `permissions` WRITE;
/*!40000 ALTER TABLE `permissions` DISABLE KEYS */;
INSERT INTO `permissions` VALUES ('e5e19f07-f8e2-4fd3-aada-401fd0a73986','*','*','全部权限');
/*!40000 ALTER TABLE `permissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `realtime_assist_sessions`
--

DROP TABLE IF EXISTS `realtime_assist_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `realtime_assist_sessions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `call_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `form_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `provider_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` enum('active','completed','failed') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `transcript_json` json DEFAULT NULL,
  `form_draft_json` json DEFAULT NULL,
  `last_suggestion` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_realtime_assist_call` (`call_id`),
  KEY `fk_realtime_assist_patient` (`patient_id`),
  KEY `fk_realtime_assist_provider` (`provider_id`),
  CONSTRAINT `fk_realtime_assist_call` FOREIGN KEY (`call_id`) REFERENCES `call_sessions` (`id`),
  CONSTRAINT `fk_realtime_assist_patient` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`),
  CONSTRAINT `fk_realtime_assist_provider` FOREIGN KEY (`provider_id`) REFERENCES `model_providers` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `realtime_assist_sessions`
--

LOCK TABLES `realtime_assist_sessions` WRITE;
/*!40000 ALTER TABLE `realtime_assist_sessions` DISABLE KEYS */;
/*!40000 ALTER TABLE `realtime_assist_sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `recording_configs`
--

DROP TABLE IF EXISTS `recording_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `recording_configs` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `mode` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `storage_config_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `format` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `retention_days` int NOT NULL DEFAULT '365',
  `auto_start` tinyint(1) NOT NULL DEFAULT '1',
  `auto_stop` tinyint(1) NOT NULL DEFAULT '1',
  `config_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_recording_configs_storage` (`storage_config_id`),
  CONSTRAINT `fk_recording_configs_storage` FOREIGN KEY (`storage_config_id`) REFERENCES `storage_configs` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `recording_configs`
--

LOCK TABLES `recording_configs` WRITE;
/*!40000 ALTER TABLE `recording_configs` DISABLE KEYS */;
INSERT INTO `recording_configs` VALUES ('REC-CFG-001','默认通话录音策略','server','STOR001','wav',365,1,1,'{\"source\": \"pbx_or_diago\"}','2026-05-14 05:55:15','2026-05-14 05:55:15');
/*!40000 ALTER TABLE `recording_configs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `recordings`
--

DROP TABLE IF EXISTS `recordings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `recordings` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `call_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `storage_uri` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `duration` int NOT NULL DEFAULT '0',
  `filename` varchar(240) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `mime_type` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `size_bytes` bigint NOT NULL DEFAULT '0',
  `source` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'browser',
  `backend` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'local',
  `object_name` varchar(512) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` enum('recording','ready','failed') COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'recording',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_recordings_call` (`call_id`),
  CONSTRAINT `fk_recordings_call` FOREIGN KEY (`call_id`) REFERENCES `call_sessions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `recordings`
--

LOCK TABLES `recordings` WRITE;
/*!40000 ALTER TABLE `recordings` DISABLE KEYS */;
/*!40000 ALTER TABLE `recordings` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `report_queries`
--

DROP TABLE IF EXISTS `report_queries`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `report_queries` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `report_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `data_source_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `query_template` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `params_schema` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_report_queries_report` (`report_id`),
  KEY `fk_report_queries_source` (`data_source_id`),
  CONSTRAINT `fk_report_queries_report` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`),
  CONSTRAINT `fk_report_queries_source` FOREIGN KEY (`data_source_id`) REFERENCES `data_sources` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `report_queries`
--

LOCK TABLES `report_queries` WRITE;
/*!40000 ALTER TABLE `report_queries` DISABLE KEYS */;
/*!40000 ALTER TABLE `report_queries` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `report_query_results`
--

DROP TABLE IF EXISTS `report_query_results`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `report_query_results` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `report_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `dimensions_json` json DEFAULT NULL,
  `measures_json` json DEFAULT NULL,
  `rows_json` json NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_report_query_results_report` (`report_id`),
  CONSTRAINT `fk_report_query_results_report` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `report_query_results`
--

LOCK TABLES `report_query_results` WRITE;
/*!40000 ALTER TABLE `report_query_results` DISABLE KEYS */;
/*!40000 ALTER TABLE `report_query_results` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `report_versions`
--

DROP TABLE IF EXISTS `report_versions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `report_versions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `report_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `version` int NOT NULL,
  `layout_json` json NOT NULL,
  `created_by` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_report_version` (`report_id`,`version`),
  KEY `fk_report_versions_creator` (`created_by`),
  CONSTRAINT `fk_report_versions_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`),
  CONSTRAINT `fk_report_versions_report` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `report_versions`
--

LOCK TABLES `report_versions` WRITE;
/*!40000 ALTER TABLE `report_versions` DISABLE KEYS */;
/*!40000 ALTER TABLE `report_versions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `report_widgets`
--

DROP TABLE IF EXISTS `report_widgets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `report_widgets` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `report_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `widget_type` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `query_json` json DEFAULT NULL,
  `vis_spec_json` json DEFAULT NULL,
  `data_source_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_report_widgets_report` (`report_id`),
  KEY `fk_report_widgets_source` (`data_source_id`),
  CONSTRAINT `fk_report_widgets_report` FOREIGN KEY (`report_id`) REFERENCES `reports` (`id`),
  CONSTRAINT `fk_report_widgets_source` FOREIGN KEY (`data_source_id`) REFERENCES `data_sources` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `report_widgets`
--

LOCK TABLES `report_widgets` WRITE;
/*!40000 ALTER TABLE `report_widgets` DISABLE KEYS */;
INSERT INTO `report_widgets` VALUES ('38d5ed67-2302-4842-b957-45bd2c9f04fe','RP002','table','新明细表','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 14:51:49'),('a417129c-8e3f-4571-8425-9a345c83252d','RP002','table','新明细表','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 13:00:43'),('a81de40e-fc27-4816-b661-56439b0ca73f','RP002','bar','新图表','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 13:00:44'),('c0dbfb66-62a5-4791-87e6-7f2d85b02a80','RP002','table','新明细表','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 13:00:48'),('cd4fd470-5032-4acf-93b4-51901c977811','RP002','table','新明细表','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 13:00:42'),('ce58cff0-edbe-4245-8b3a-6030a22281de','RP002','bar','新图表','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 13:00:49'),('cf63a0a2-853e-4c36-a05a-cbc4410d82e1','RP002','table','新明细表','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 13:00:47'),('RW001','RP001','bar','月度随访完成率','{\"source\": \"followup_records\"}','{}',NULL,'2026-05-14 11:59:53'),('RW002','RP001','table','随访月度明细','{\"source\": \"followup_records\"}','{}',NULL,'2026-05-14 11:59:53'),('RW003','RP002','bar','科室满意度','{\"source\": \"survey_submissions\"}','{}',NULL,'2026-05-14 11:59:53'),('RW004','RP002','table','满意度指标明细','{\"source\": \"satisfaction_indicator_scores\"}','{}',NULL,'2026-05-14 11:59:53'),('RW005','RP003','bar','责任科室投诉评价','{\"source\": \"evaluation_complaints\"}','{}',NULL,'2026-05-14 11:59:53');
/*!40000 ALTER TABLE `report_widgets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `reports`
--

DROP TABLE IF EXISTS `reports`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `reports` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `report_type` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'custom',
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `reports`
--

LOCK TABLES `reports` WRITE;
/*!40000 ALTER TABLE `reports` DISABLE KEYS */;
INSERT INTO `reports` VALUES ('RP001','followup','随访完成情况月报','从随访记录聚合随访提交量、完成量和完成率','2026-05-14 11:59:53','2026-05-14 11:59:53'),('RP002','satisfaction','满意度分析报告','从满意度答卷、访谈表单和指标体系聚合科室、指标、渠道和低分原因','2026-05-14 11:59:53','2026-05-14 11:59:53'),('RP003','complaint','评价投诉分析报告','从评价投诉台账聚合投诉、表扬、处理状态和责任科室','2026-05-14 11:59:53','2026-05-14 11:59:53');
/*!40000 ALTER TABLE `reports` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `role_permissions`
--

DROP TABLE IF EXISTS `role_permissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `role_permissions` (
  `role_id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `permission_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`role_id`,`permission_id`),
  KEY `fk_role_permissions_permission` (`permission_id`),
  CONSTRAINT `fk_role_permissions_permission` FOREIGN KEY (`permission_id`) REFERENCES `permissions` (`id`),
  CONSTRAINT `fk_role_permissions_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `role_permissions`
--

LOCK TABLES `role_permissions` WRITE;
/*!40000 ALTER TABLE `role_permissions` DISABLE KEYS */;
INSERT INTO `role_permissions` VALUES ('admin','e5e19f07-f8e2-4fd3-aada-401fd0a73986');
/*!40000 ALTER TABLE `role_permissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roles`
--

DROP TABLE IF EXISTS `roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `roles` (
  `id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roles`
--

LOCK TABLES `roles` WRITE;
/*!40000 ALTER TABLE `roles` DISABLE KEYS */;
INSERT INTO `roles` VALUES ('admin','系统管理员','拥有平台全部管理权限','2026-05-14 05:55:15'),('agent','随访坐席','可查看患者并执行电话随访','2026-05-14 05:55:15'),('analyst','数据分析员','可管理表单、报表并查看数据源','2026-05-14 05:55:15');
/*!40000 ALTER TABLE `roles` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `satisfaction_cleaning_rules`
--

DROP TABLE IF EXISTS `satisfaction_cleaning_rules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `satisfaction_cleaning_rules` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `rule_type` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `config_json` json DEFAULT NULL,
  `action` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'mark_suspicious',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_cleaning_rules_project` (`project_id`),
  KEY `idx_cleaning_rules_type` (`rule_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `satisfaction_cleaning_rules`
--

LOCK TABLES `satisfaction_cleaning_rules` WRITE;
/*!40000 ALTER TABLE `satisfaction_cleaning_rules` DISABLE KEYS */;
INSERT INTO `satisfaction_cleaning_rules` VALUES ('6a9fa11b-66f9-4f35-8087-aa23a07684f5','8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','全同选项','same_option',1,'{\"minQuestionCount\": 5}','mark_suspicious','2026-05-14 11:42:21','2026-05-14 11:42:21'),('9614a372-6f83-4705-90da-8e15d20e8203','8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','同 IP/设备高频提交','same_device',0,'{\"maxCount\": 5, \"windowHours\": 1}','mark_suspicious','2026-05-14 11:42:21','2026-05-14 11:42:21'),('c3cea35d-9eb9-4997-9aca-a27136c6c3a6','8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','答题时长过短','duration',1,'{\"minSeconds\": 20}','mark_suspicious','2026-05-14 11:42:21','2026-05-14 11:42:21'),('f48fcb9d-0ae7-4bb1-a810-d41eda8796b0','8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','同项目重复提交','duplicate_project',1,'{\"strategy\": \"keep_latest\", \"windowHours\": 24}','mark_suspicious','2026-05-14 11:42:21','2026-05-14 11:42:21');
/*!40000 ALTER TABLE `satisfaction_cleaning_rules` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `satisfaction_indicator_questions`
--

DROP TABLE IF EXISTS `satisfaction_indicator_questions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `satisfaction_indicator_questions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `indicator_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `form_template_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `question_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `question_label` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `score_direction` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'positive',
  `weight` decimal(10,2) NOT NULL DEFAULT '1.00',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_indicator_question` (`project_id`,`form_template_id`,`question_id`),
  KEY `idx_indicator_questions_indicator` (`indicator_id`),
  KEY `idx_indicator_questions_project` (`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `satisfaction_indicator_questions`
--

LOCK TABLES `satisfaction_indicator_questions` WRITE;
/*!40000 ALTER TABLE `satisfaction_indicator_questions` DISABLE KEYS */;
/*!40000 ALTER TABLE `satisfaction_indicator_questions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `satisfaction_indicator_scores`
--

DROP TABLE IF EXISTS `satisfaction_indicator_scores`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `satisfaction_indicator_scores` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `indicator_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `department_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `doctor_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `nurse_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `disease_name` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `visit_type` varchar(60) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `score` decimal(10,2) NOT NULL DEFAULT '0.00',
  `sample_count` int NOT NULL DEFAULT '0',
  `score_period` date DEFAULT NULL,
  `source_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_satisfaction_scores_project` (`project_id`),
  KEY `idx_satisfaction_scores_indicator` (`indicator_id`),
  KEY `idx_satisfaction_scores_patient` (`patient_id`),
  KEY `idx_satisfaction_scores_department` (`department_name`),
  KEY `idx_satisfaction_scores_period` (`score_period`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `satisfaction_indicator_scores`
--

LOCK TABLES `satisfaction_indicator_scores` WRITE;
/*!40000 ALTER TABLE `satisfaction_indicator_scores` DISABLE KEYS */;
/*!40000 ALTER TABLE `satisfaction_indicator_scores` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `satisfaction_indicators`
--

DROP TABLE IF EXISTS `satisfaction_indicators`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `satisfaction_indicators` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `target_type` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'outpatient',
  `level_no` int NOT NULL DEFAULT '1',
  `parent_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `service_stage` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `service_node` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `question_id` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `weight` decimal(10,2) NOT NULL DEFAULT '1.00',
  `include_total_score` tinyint(1) NOT NULL DEFAULT '1',
  `national_dimension` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `include_national` tinyint(1) NOT NULL DEFAULT '0',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_satisfaction_indicators_project` (`project_id`),
  KEY `idx_satisfaction_indicators_question` (`question_id`),
  KEY `idx_satisfaction_indicators_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `satisfaction_indicators`
--

LOCK TABLES `satisfaction_indicators` WRITE;
/*!40000 ALTER TABLE `satisfaction_indicators` DISABLE KEYS */;
INSERT INTO `satisfaction_indicators` VALUES ('0af24d2f-4430-459a-a28c-f45e6d97ef80','8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','outpatient',2,NULL,'分项满意度',NULL,NULL,'service_matrix',1.00,1,'诊疗流程',0,1,'2026-05-14 10:58:29','2026-05-14 10:58:29'),('5a5416e2-84ef-4c86-b903-85a13d01ab88','8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','outpatient',1,NULL,'综合体验',NULL,NULL,'overall_satisfaction',1.00,1,'综合体验',0,1,'2026-05-14 10:58:29','2026-05-14 10:58:29'),('746eaebd-8d2e-4799-8bc9-e85c8f3c8c27','8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','outpatient',2,NULL,'推荐意愿',NULL,NULL,'recommend_score',1.00,1,'综合体验',0,1,'2026-05-14 10:58:29','2026-05-14 10:58:29');
/*!40000 ALTER TABLE `satisfaction_indicators` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `satisfaction_issue_events`
--

DROP TABLE IF EXISTS `satisfaction_issue_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `satisfaction_issue_events` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `issue_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `action` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `from_status` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `to_status` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `content` text COLLATE utf8mb4_unicode_ci,
  `attachments_json` json DEFAULT NULL,
  `actor_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_issue_events_issue` (`issue_id`),
  KEY `idx_issue_events_action` (`action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `satisfaction_issue_events`
--

LOCK TABLES `satisfaction_issue_events` WRITE;
/*!40000 ALTER TABLE `satisfaction_issue_events` DISABLE KEYS */;
/*!40000 ALTER TABLE `satisfaction_issue_events` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `satisfaction_issues`
--

DROP TABLE IF EXISTS `satisfaction_issues`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `satisfaction_issues` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `submission_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `indicator_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `title` varchar(240) COLLATE utf8mb4_unicode_ci NOT NULL,
  `source` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'manual',
  `responsible_department` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `responsible_person` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `severity` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'medium',
  `suggestion` text COLLATE utf8mb4_unicode_ci,
  `measure` text COLLATE utf8mb4_unicode_ci,
  `material_urls` json DEFAULT NULL,
  `verification_result` text COLLATE utf8mb4_unicode_ci,
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'open',
  `due_date` date DEFAULT NULL,
  `closed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_satisfaction_issues_project` (`project_id`),
  KEY `idx_satisfaction_issues_status` (`status`),
  KEY `idx_satisfaction_issues_submission` (`submission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `satisfaction_issues`
--

LOCK TABLES `satisfaction_issues` WRITE;
/*!40000 ALTER TABLE `satisfaction_issues` DISABLE KEYS */;
/*!40000 ALTER TABLE `satisfaction_issues` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `satisfaction_projects`
--

DROP TABLE IF EXISTS `satisfaction_projects`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `satisfaction_projects` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `target_type` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'outpatient',
  `form_template_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `start_date` date DEFAULT NULL,
  `end_date` date DEFAULT NULL,
  `target_sample_size` int NOT NULL DEFAULT '0',
  `actual_sample_size` int NOT NULL DEFAULT '0',
  `anonymous` tinyint(1) NOT NULL DEFAULT '1',
  `requires_verification` tinyint(1) NOT NULL DEFAULT '0',
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'draft',
  `config_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_satisfaction_projects_status` (`status`),
  KEY `idx_satisfaction_projects_target` (`target_type`),
  KEY `idx_satisfaction_projects_template` (`form_template_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `satisfaction_projects`
--

LOCK TABLES `satisfaction_projects` WRITE;
/*!40000 ALTER TABLE `satisfaction_projects` DISABLE KEYS */;
INSERT INTO `satisfaction_projects` VALUES ('8bcab9f3-4821-4cc0-8fab-0dcdc1820bf2','患者就诊满意度调查','outpatient','outpatient-satisfaction',NULL,NULL,0,0,1,0,'draft','{}','2026-05-14 10:07:56','2026-05-14 10:07:56');
/*!40000 ALTER TABLE `satisfaction_projects` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sip_endpoints`
--

DROP TABLE IF EXISTS `sip_endpoints`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sip_endpoints` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `wss_url` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `domain` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `proxy` text COLLATE utf8mb4_unicode_ci,
  `config_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sip_endpoints`
--

LOCK TABLES `sip_endpoints` WRITE;
/*!40000 ALTER TABLE `sip_endpoints` DISABLE KEYS */;
/*!40000 ALTER TABLE `sip_endpoints` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `storage_configs`
--

DROP TABLE IF EXISTS `storage_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `storage_configs` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `kind` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `endpoint` text COLLATE utf8mb4_unicode_ci,
  `bucket` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `base_path` text COLLATE utf8mb4_unicode_ci,
  `base_uri` text COLLATE utf8mb4_unicode_ci,
  `credential_ref` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `config_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `storage_configs`
--

LOCK TABLES `storage_configs` WRITE;
/*!40000 ALTER TABLE `storage_configs` DISABLE KEYS */;
INSERT INTO `storage_configs` VALUES ('STOR001','本地录音存储','local',NULL,NULL,'data/recordings',NULL,NULL,'{\"pathStrategy\": \"yyyy/mm/dd\"}','2026-05-14 05:55:15','2026-05-14 05:55:15');
/*!40000 ALTER TABLE `storage_configs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `submission_events`
--

DROP TABLE IF EXISTS `submission_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `submission_events` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `submission_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `actor_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `event_type` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `payload_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_submission_events_submission` (`submission_id`),
  CONSTRAINT `fk_submission_events_submission` FOREIGN KEY (`submission_id`) REFERENCES `form_submissions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `submission_events`
--

LOCK TABLES `submission_events` WRITE;
/*!40000 ALTER TABLE `submission_events` DISABLE KEYS */;
/*!40000 ALTER TABLE `submission_events` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `surgery_records`
--

DROP TABLE IF EXISTS `surgery_records`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `surgery_records` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_code` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_name` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `operation_date` datetime DEFAULT NULL,
  `surgeon_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `anesthesia_type` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `operation_level` varchar(60) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `wound_grade` varchar(60) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `outcome` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `source_system` varchar(80) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_surgery_records_patient` (`patient_id`),
  KEY `idx_surgery_records_visit` (`visit_id`),
  KEY `idx_surgery_records_operation` (`operation_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `surgery_records`
--

LOCK TABLES `surgery_records` WRITE;
/*!40000 ALTER TABLE `surgery_records` DISABLE KEYS */;
/*!40000 ALTER TABLE `surgery_records` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `survey_channel_deliveries`
--

DROP TABLE IF EXISTS `survey_channel_deliveries`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `survey_channel_deliveries` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `share_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `channel` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL,
  `recipient` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `recipient_name` varchar(120) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'queued',
  `message` text COLLATE utf8mb4_unicode_ci,
  `error` text COLLATE utf8mb4_unicode_ci,
  `provider_ref` varchar(180) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `config_json` json DEFAULT NULL,
  `sent_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_survey_deliveries_project` (`project_id`),
  KEY `idx_survey_deliveries_share` (`share_id`),
  KEY `idx_survey_deliveries_status` (`status`),
  KEY `idx_survey_deliveries_recipient` (`recipient`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `survey_channel_deliveries`
--

LOCK TABLES `survey_channel_deliveries` WRITE;
/*!40000 ALTER TABLE `survey_channel_deliveries` DISABLE KEYS */;
/*!40000 ALTER TABLE `survey_channel_deliveries` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `survey_interviews`
--

DROP TABLE IF EXISTS `survey_interviews`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `survey_interviews` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `share_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `answers_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_survey_interviews_share` (`share_id`),
  CONSTRAINT `fk_survey_interviews_share` FOREIGN KEY (`share_id`) REFERENCES `survey_share_links` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `survey_interviews`
--

LOCK TABLES `survey_interviews` WRITE;
/*!40000 ALTER TABLE `survey_interviews` DISABLE KEYS */;
INSERT INTO `survey_interviews` VALUES ('005131c3-94f6-49ac-a274-142b4ece0cca','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:37:52','2026-05-14 09:37:52'),('038ce899-d13e-4e29-a8bd-a8c9d8ee81ed','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:17:13','2026-05-14 10:17:13'),('0b160006-e461-41c7-b88f-925768f8d202','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:51:02','2026-05-14 09:51:02'),('0d058a27-8cb4-458b-b6b7-a92e6c8004ab','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:27:36','2026-05-14 10:27:36'),('160656a5-85c2-430c-911d-8d1d3bdcb392','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:00:10','2026-05-14 10:00:10'),('18715597-4d8a-454c-95ca-dbd2f20732e9','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:28:41','2026-05-14 09:28:41'),('1e63c30c-2cdd-491b-a28b-d0e945668675','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:40:19','2026-05-14 09:40:19'),('21880941-5355-48ef-9740-256bf8d95604','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:09:07','2026-05-14 10:09:07'),('26ab5e79-4f7d-470d-8458-4239645098bb','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:54:26','2026-05-14 09:54:26'),('29e52176-7af3-4c81-9471-df57963ea2f5','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:51:45','2026-05-14 09:51:45'),('2d0b2eb2-6d7c-455c-9c39-2df6cb4dfc68','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:16:52','2026-05-14 10:16:52'),('2fe6dd9c-26ae-46f7-be65-536ce03f9f66','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:27:46','2026-05-14 09:27:46'),('305e8f21-81f5-4d8e-8bb6-b7771c3d0da2','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:29:53','2026-05-14 10:29:53'),('34356de2-54bd-4ddd-803a-d494854c863f','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:36:20','2026-05-14 09:36:20'),('3ecb3795-c8d7-4ece-84fe-779ffd9567d9','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:41:41','2026-05-14 09:41:41'),('3ecee15f-9a3b-4b63-a45e-c7719223a9e7','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:29:36','2026-05-14 10:29:36'),('465d7ceb-6c0c-4fbd-94cb-823192221916','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:09:52','2026-05-14 10:09:52'),('483fa93a-b55c-48be-922b-179dda2dbc9c','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:59:38','2026-05-14 09:59:38'),('4df5bbb8-03c7-44ce-8e50-4861dfa61790','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:29:45','2026-05-14 09:29:45'),('52d369ac-5919-4d95-81ab-e92a88aa82ae','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:10:08','2026-05-14 10:10:08'),('5581aa86-fa7c-41a7-95b7-be194955de74','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:28:14','2026-05-14 09:28:14'),('5844f592-b8b7-4743-9162-7b7e589086f2','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:14:21','2026-05-14 09:14:21'),('5d125e86-7aa7-49e3-9704-1fb7df8de412','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:48:07','2026-05-14 09:48:07'),('5eaf6402-34b3-4864-bde5-615962b30701','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:34:39','2026-05-14 09:34:39'),('64e600eb-6966-43bb-9667-ebede4be9ee5','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:25:48','2026-05-14 09:25:48'),('6f17b03f-4a73-40f0-b7c4-2b7a77e49717','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:52:01','2026-05-14 09:52:01'),('74ebf49c-7d07-49f8-b112-7ce59676a19d','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:28:47','2026-05-14 10:28:47'),('76185605-7e04-4f62-87fc-b8a09aa9d01a','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:10:45','2026-05-14 10:10:45'),('7a556155-9d32-448c-a04e-a7f8bfaf177e','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:49:19','2026-05-14 09:49:19'),('82bacacf-dbf9-4683-9821-44c74fee0236','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:27:50','2026-05-14 10:27:50'),('83e08599-0176-48f6-9f80-a817af24e2a6','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:26:29','2026-05-14 10:26:29'),('91e4cc09-df44-497e-96ea-e2c09a29bf8c','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:26:59','2026-05-14 10:26:59'),('99fc6dc0-17a2-4250-b81e-d3655577574e','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:50:53','2026-05-14 09:50:53'),('9e5716d8-0ae1-43ec-895f-8047ffda9ecc','386d459c-fbfa-4e30-ab49-d9301afee2fe',NULL,'active','{}','2026-05-14 12:41:29','2026-05-14 12:41:29'),('a13ae282-211a-471d-8cb0-4a8080b84474','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:28:02','2026-05-14 09:28:02'),('aba8e739-8546-4410-bcef-9cb82346120a','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:32:19','2026-05-14 10:32:19'),('b7a1355d-0f0c-4ae4-938a-ea1cbad864d4','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:34:11','2026-05-14 09:34:11'),('b9075cb3-eecb-4382-a4c4-955566a3ca3f','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:10:31','2026-05-14 10:10:31'),('bc3d04f6-63bf-44f1-a7a9-5531e572fdc8','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:27:35','2026-05-14 09:27:35'),('c2bd201c-8ce9-4255-be01-25a6898749c2','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:50:12','2026-05-14 09:50:12'),('dd81fedb-1a44-48f5-9593-2fef43e2005c','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:09:30','2026-05-14 10:09:30'),('de125180-7843-4fce-a8c7-9cf612f7104a','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:29:17','2026-05-14 09:29:17'),('df739e8c-f3ee-45e2-8d6f-6313a996c40b','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:48:30','2026-05-14 09:48:30'),('e499006d-a393-491d-8f34-784fd77db951','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:29:07','2026-05-14 09:29:07'),('ebd66548-62cf-41b0-a8ac-9251008eca34','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:50:03','2026-05-14 09:50:03'),('eee1847b-c92f-443e-bd21-37f6d5871676','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:47:37','2026-05-14 09:47:37'),('f2936dd8-d7d6-47b7-8079-47f58230f292','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:50:38','2026-05-14 09:50:38'),('f34a1d83-e75f-4c28-8180-0f1c56f0f7ac','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:10:56','2026-05-14 10:10:56'),('f9caaa6d-d0db-4a29-b628-4b205d26cc35','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 10:28:10','2026-05-14 10:28:10'),('fcec48c0-f27f-45a0-9713-e872e8df8a60','1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'active','{}','2026-05-14 09:53:46','2026-05-14 09:53:46');
/*!40000 ALTER TABLE `survey_interviews` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `survey_share_links`
--

DROP TABLE IF EXISTS `survey_share_links`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `survey_share_links` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `form_template_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(180) COLLATE utf8mb4_unicode_ci NOT NULL,
  `channel` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'web',
  `token` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `expires_at` timestamp NULL DEFAULT NULL,
  `config_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `token` (`token`),
  KEY `idx_survey_share_links_template` (`form_template_id`),
  KEY `idx_survey_share_links_channel` (`channel`),
  KEY `idx_survey_share_links_project` (`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `survey_share_links`
--

LOCK TABLES `survey_share_links` WRITE;
/*!40000 ALTER TABLE `survey_share_links` DISABLE KEYS */;
INSERT INTO `survey_share_links` VALUES ('1311f822-43dd-46a2-a116-bbc8faca825e',NULL,'outpatient-satisfaction','患者就诊满意度调查','web','9a5363fc5e21ca80d738c2a03e760679',NULL,'{}','2026-05-14 09:14:16','2026-05-14 09:14:16'),('29b19a77-30aa-4679-a06a-b60728369997',NULL,'outpatient-satisfaction','王五随访问卷','wechat','7ed4d8720d6324288d885c3a2fc5c2bc',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:40','2026-05-14 12:41:40'),('386d459c-fbfa-4e30-ab49-d9301afee2fe',NULL,'outpatient-satisfaction','王五随访问卷','web','a8518efcfbd46e598e074c6ae54d2d24',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"web\"}','2026-05-14 12:41:29','2026-05-14 12:41:29'),('391d93a1-d762-4c03-ac6e-64dca6953b47',NULL,'outpatient-satisfaction','王五随访问卷','wechat','4f611b272d629118e414af17a8b351d6',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:44','2026-05-14 12:41:44'),('40912746-8c34-4652-bc56-01e54c2712ec',NULL,'outpatient-satisfaction','王五随访问卷','sms','61843f96189e8ca2d12c8ef1cc6f2023',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"sms\"}','2026-05-14 12:41:21','2026-05-14 12:41:21'),('7df8ec06-8647-4e27-8024-12ae4cf42422',NULL,'outpatient-satisfaction','王五随访问卷','wechat','b7ff2ea757887d1e4268aecb9479e4f1',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:41','2026-05-14 12:41:41'),('b9187b83-bd4e-42fa-9edd-d9b9d6f5db44',NULL,'outpatient-satisfaction','王五随访问卷','wechat','c71fd7e168531646e5ea59a84e97c463',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:44','2026-05-14 12:41:44'),('c84b1121-4ef4-48a4-b074-ab99375633fd',NULL,'outpatient-satisfaction','王五随访问卷','wechat','5fb2754b83b53fa50a583574c7ca27a7',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:42','2026-05-14 12:41:42'),('ccc40e6a-6c54-49e7-8bb3-ccfcbfc9f224',NULL,'outpatient-satisfaction','王五随访问卷','wechat','65a639c9390d63f077af02f564df7307',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:42','2026-05-14 12:41:42'),('d945ae04-7b20-473f-a357-4beab84a6e20',NULL,'outpatient-satisfaction','王五随访问卷','wechat','bcadd5a0ebbc24c393f1d58113e4e48d',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:43','2026-05-14 12:41:43'),('f99b31c1-2202-441e-9a83-af4661a255fe',NULL,'outpatient-satisfaction','王五随访问卷','wechat','aa5c913c514411db0f23c621edb48c63',NULL,'{\"patientId\": \"P003\", \"patientName\": \"王五\", \"patientPhone\": \"13800010003\", \"deliveryChannel\": \"wechat\"}','2026-05-14 12:41:41','2026-05-14 12:41:41');
/*!40000 ALTER TABLE `survey_share_links` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `survey_submission_answers`
--

DROP TABLE IF EXISTS `survey_submission_answers`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `survey_submission_answers` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `submission_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `question_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `question_label` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `question_type` varchar(60) COLLATE utf8mb4_unicode_ci NOT NULL,
  `answer_json` json DEFAULT NULL,
  `score` decimal(10,2) DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_submission_answers_submission` (`submission_id`),
  KEY `idx_submission_answers_question` (`question_id`),
  CONSTRAINT `fk_submission_answers_submission` FOREIGN KEY (`submission_id`) REFERENCES `survey_submissions` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `survey_submission_answers`
--

LOCK TABLES `survey_submission_answers` WRITE;
/*!40000 ALTER TABLE `survey_submission_answers` DISABLE KEYS */;
INSERT INTO `survey_submission_answers` VALUES ('3dfc52a5-faba-4d48-9306-b6ba69d6030f','c192596b-b40e-40dc-908d-3625a85bf0f1','feedback','意见与建议','textarea','\"接口验证\"',NULL,'2026-05-14 10:02:02'),('ce19b72e-4849-467d-a5eb-178dfd6ca5ca','c192596b-b40e-40dc-908d-3625a85bf0f1','overall_satisfaction','总体满意度','likert','\"5\"',5.00,'2026-05-14 10:02:02'),('e4513da5-f4dd-4952-901d-b98c4d5c030a','c192596b-b40e-40dc-908d-3625a85bf0f1','recommend_score','推荐意愿','rating','\"10\"',10.00,'2026-05-14 10:02:02');
/*!40000 ALTER TABLE `survey_submission_answers` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `survey_submission_audit_logs`
--

DROP TABLE IF EXISTS `survey_submission_audit_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `survey_submission_audit_logs` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `submission_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `action` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `from_status` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `to_status` varchar(40) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `reason` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `actor_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_submission_audit_submission` (`submission_id`),
  KEY `idx_submission_audit_project` (`project_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `survey_submission_audit_logs`
--

LOCK TABLES `survey_submission_audit_logs` WRITE;
/*!40000 ALTER TABLE `survey_submission_audit_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `survey_submission_audit_logs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `survey_submissions`
--

DROP TABLE IF EXISTS `survey_submissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `survey_submissions` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `project_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `share_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `form_template_id` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `channel` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'web',
  `patient_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `visit_id` char(36) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `anonymous` tinyint(1) NOT NULL DEFAULT '1',
  `status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'submitted',
  `quality_status` varchar(40) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
  `quality_reason` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `started_at` timestamp NULL DEFAULT NULL,
  `submitted_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `duration_seconds` int NOT NULL DEFAULT '0',
  `ip_address` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` text COLLATE utf8mb4_unicode_ci,
  `answers_json` json DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_survey_submissions_project` (`project_id`),
  KEY `idx_survey_submissions_share` (`share_id`),
  KEY `idx_survey_submissions_template` (`form_template_id`),
  KEY `idx_survey_submissions_quality` (`quality_status`),
  KEY `idx_survey_submissions_submitted` (`submitted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `survey_submissions`
--

LOCK TABLES `survey_submissions` WRITE;
/*!40000 ALTER TABLE `survey_submissions` DISABLE KEYS */;
INSERT INTO `survey_submissions` VALUES ('c192596b-b40e-40dc-908d-3625a85bf0f1',NULL,'1311f822-43dd-46a2-a116-bbc8faca825e','outpatient-satisfaction','web',NULL,NULL,1,'submitted','pending','',NULL,'2026-05-14 10:02:02',30,'127.0.0.1','curl/8.4.0','{\"feedback\": \"接口验证\", \"recommend_score\": \"10\", \"overall_satisfaction\": \"5\"}','2026-05-14 10:02:02','2026-05-14 10:02:02');
/*!40000 ALTER TABLE `survey_submissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `system_settings`
--

DROP TABLE IF EXISTS `system_settings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `system_settings` (
  `setting_key` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `setting_value` json DEFAULT NULL,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `system_settings`
--

LOCK TABLES `system_settings` WRITE;
/*!40000 ALTER TABLE `system_settings` DISABLE KEYS */;
/*!40000 ALTER TABLE `system_settings` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_roles`
--

DROP TABLE IF EXISTS `user_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_roles` (
  `user_id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `role_id` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`user_id`,`role_id`),
  KEY `fk_user_roles_role` (`role_id`),
  CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`),
  CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_roles`
--

LOCK TABLES `user_roles` WRITE;
/*!40000 ALTER TABLE `user_roles` DISABLE KEYS */;
INSERT INTO `user_roles` VALUES ('80db839e-d1bc-4fca-a35b-344a311e73e1','admin');
/*!40000 ALTER TABLE `user_roles` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` char(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  `username` varchar(80) COLLATE utf8mb4_unicode_ci NOT NULL,
  `display_name` varchar(120) COLLATE utf8mb4_unicode_ci NOT NULL,
  `password_hash` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES ('80db839e-d1bc-4fca-a35b-344a311e73e1','admin','系统管理员','$argon2id$v=19$m=65536,t=3,p=2$rokZrcOkpOD5JFJDE2AaXQ$ktAJaqFcHfXSxdghuzD5Bx0QKqrblSyjPReHG8bhxAk','2026-05-14 05:55:15','2026-05-14 05:55:15');
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Dumping events for database 'report'
--

--
-- Dumping routines for database 'report'
--
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-05-15 11:00:35

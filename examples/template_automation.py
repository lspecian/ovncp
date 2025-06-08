#!/usr/bin/env python3
"""
OVN Control Platform - Policy Template Automation Example

This script demonstrates how to automate network policy deployment
using templates for a complete infrastructure setup.
"""

import json
import requests
from typing import Dict, List, Any
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class OVNCPTemplateClient:
    """Client for OVN Control Platform template operations"""
    
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url.rstrip('/')
        self.headers = {
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        }
        
    def list_templates(self, category: str = None, tags: List[str] = None) -> List[Dict]:
        """List available templates"""
        params = {}
        if category:
            params['category'] = category
        if tags:
            params['tag'] = tags
            
        response = requests.get(
            f"{self.base_url}/api/v1/templates",
            headers=self.headers,
            params=params
        )
        response.raise_for_status()
        return response.json()['templates']
    
    def validate_template(self, template_id: str, variables: Dict[str, Any]) -> Dict:
        """Validate template with variables"""
        data = {
            'template_id': template_id,
            'variables': variables
        }
        
        response = requests.post(
            f"{self.base_url}/api/v1/templates/validate",
            headers=self.headers,
            json=data
        )
        response.raise_for_status()
        return response.json()
    
    def instantiate_template(self, template_id: str, variables: Dict[str, Any], 
                           target_switch: str = None, dry_run: bool = False) -> Dict:
        """Instantiate a template"""
        data = {
            'template_id': template_id,
            'variables': variables,
            'dry_run': dry_run
        }
        
        if target_switch:
            data['target_switch'] = target_switch
            
        response = requests.post(
            f"{self.base_url}/api/v1/templates/instantiate",
            headers=self.headers,
            json=data
        )
        response.raise_for_status()
        return response.json()


class InfrastructureDeployer:
    """Deploy complete infrastructure using templates"""
    
    def __init__(self, client: OVNCPTemplateClient):
        self.client = client
        
    def deploy_web_tier(self, servers: List[Dict[str, str]], switch: str):
        """Deploy web server policies"""
        logger.info(f"Deploying web tier on switch {switch}")
        
        for server in servers:
            logger.info(f"Configuring web server {server['name']} ({server['ip']})")
            
            # Validate first
            validation = self.client.validate_template('web-server', {
                'server_ip': server['ip'],
                'allowed_sources': server.get('allowed_sources', '0.0.0.0/0'),
                'enable_ssh': server.get('enable_ssh', True),
                'ssh_sources': server.get('ssh_sources', '10.0.100.0/24')
            })
            
            if not validation['valid']:
                logger.error(f"Validation failed for {server['name']}: {validation['errors']}")
                continue
                
            # Deploy
            result = self.client.instantiate_template(
                'web-server',
                variables={
                    'server_ip': server['ip'],
                    'allowed_sources': server.get('allowed_sources', '0.0.0.0/0'),
                    'enable_ssh': server.get('enable_ssh', True),
                    'ssh_sources': server.get('ssh_sources', '10.0.100.0/24')
                },
                target_switch=switch
            )
            
            logger.info(f"Created {len(result['instance']['rules'])} rules for {server['name']}")
    
    def deploy_microservices(self, services: List[Dict[str, Any]], switch: str):
        """Deploy microservice mesh policies"""
        logger.info(f"Deploying microservices on switch {switch}")
        
        # Build service dependency map
        service_ips = {s['name']: s['ip'] for s in services}
        
        for service in services:
            # Convert dependency names to IPs
            allowed_ips = []
            for dep in service.get('dependencies', []):
                if dep in service_ips:
                    allowed_ips.append(service_ips[dep])
                    
            logger.info(f"Configuring service {service['name']} with dependencies: {service.get('dependencies', [])}")
            
            result = self.client.instantiate_template(
                'microservice',
                variables={
                    'service_name': service['name'],
                    'service_ip': service['ip'],
                    'service_port': service.get('port', 8080),
                    'allowed_services': ','.join(allowed_ips),
                    'monitoring_subnet': '10.0.200.0/24'
                },
                target_switch=switch
            )
            
            logger.info(f"Created {len(result['instance']['rules'])} rules for {service['name']}")
    
    def deploy_databases(self, databases: List[Dict[str, Any]], switch: str):
        """Deploy database policies"""
        logger.info(f"Deploying databases on switch {switch}")
        
        for db in databases:
            logger.info(f"Configuring database {db['name']} ({db['type']})")
            
            # Set default port based on DB type
            default_ports = {
                'mysql': 3306,
                'postgresql': 5432,
                'mongodb': 27017,
                'redis': 6379
            }
            
            port = db.get('port', default_ports.get(db['type'], 3306))
            
            result = self.client.instantiate_template(
                'database-server',
                variables={
                    'db_ip': db['ip'],
                    'db_port': port,
                    'app_subnet': db['app_subnet'],
                    'backup_server': db.get('backup_server'),
                    'enable_replication': db.get('enable_replication', False),
                    'replica_ips': ','.join(db.get('replicas', []))
                },
                target_switch=switch
            )
            
            logger.info(f"Created {len(result['instance']['rules'])} rules for {db['name']}")
    
    def deploy_zero_trust_resources(self, resources: List[Dict[str, Any]], switch: str):
        """Deploy zero-trust policies for sensitive resources"""
        logger.info(f"Deploying zero-trust policies on switch {switch}")
        
        for resource in resources:
            logger.info(f"Securing resource {resource['name']} with zero-trust policy")
            
            result = self.client.instantiate_template(
                'zero-trust',
                variables={
                    'resource_ip': resource['ip'],
                    'resource_port': resource['port'],
                    'authorized_users': ','.join(resource['authorized_users']),
                    'require_encryption': resource.get('require_encryption', True)
                },
                target_switch=switch
            )
            
            logger.info(f"Created {len(result['instance']['rules'])} rules for {resource['name']}")


def main():
    """Main deployment example"""
    
    # Configuration
    OVNCP_URL = "http://localhost:8080"
    OVNCP_TOKEN = "your-token-here"
    
    # Initialize client
    client = OVNCPTemplateClient(OVNCP_URL, OVNCP_TOKEN)
    deployer = InfrastructureDeployer(client)
    
    # Infrastructure definition
    infrastructure = {
        'web_tier': {
            'switch': 'ls-web',
            'servers': [
                {
                    'name': 'web-1',
                    'ip': '10.0.1.10',
                    'allowed_sources': '0.0.0.0/0'
                },
                {
                    'name': 'web-2',
                    'ip': '10.0.1.11',
                    'allowed_sources': '0.0.0.0/0'
                }
            ]
        },
        'app_tier': {
            'switch': 'ls-app',
            'services': [
                {
                    'name': 'api-gateway',
                    'ip': '10.0.2.10',
                    'port': 8080,
                    'dependencies': ['user-service', 'order-service']
                },
                {
                    'name': 'user-service',
                    'ip': '10.0.2.11',
                    'port': 8081,
                    'dependencies': ['auth-service']
                },
                {
                    'name': 'order-service',
                    'ip': '10.0.2.12',
                    'port': 8082,
                    'dependencies': ['user-service', 'payment-service']
                },
                {
                    'name': 'payment-service',
                    'ip': '10.0.2.13',
                    'port': 8083,
                    'dependencies': []
                },
                {
                    'name': 'auth-service',
                    'ip': '10.0.2.14',
                    'port': 8084,
                    'dependencies': []
                }
            ]
        },
        'data_tier': {
            'switch': 'ls-data',
            'databases': [
                {
                    'name': 'users-db',
                    'type': 'postgresql',
                    'ip': '10.0.3.10',
                    'app_subnet': '10.0.2.0/24',
                    'backup_server': '10.0.100.50',
                    'enable_replication': True,
                    'replicas': ['10.0.3.11', '10.0.3.12']
                },
                {
                    'name': 'orders-db',
                    'type': 'mysql',
                    'ip': '10.0.3.20',
                    'app_subnet': '10.0.2.0/24',
                    'backup_server': '10.0.100.50'
                },
                {
                    'name': 'cache',
                    'type': 'redis',
                    'ip': '10.0.3.30',
                    'app_subnet': '10.0.2.0/24'
                }
            ]
        },
        'secure_resources': {
            'switch': 'ls-secure',
            'resources': [
                {
                    'name': 'admin-panel',
                    'ip': '10.0.5.10',
                    'port': 443,
                    'authorized_users': ['10.0.100.10', '10.0.100.11'],
                    'require_encryption': True
                },
                {
                    'name': 'monitoring',
                    'ip': '10.0.5.20',
                    'port': 3000,
                    'authorized_users': ['10.0.100.10', '10.0.100.11', '10.0.100.12'],
                    'require_encryption': True
                }
            ]
        }
    }
    
    try:
        # Deploy all tiers
        logger.info("Starting infrastructure deployment...")
        
        # 1. Web tier
        deployer.deploy_web_tier(
            infrastructure['web_tier']['servers'],
            infrastructure['web_tier']['switch']
        )
        
        # 2. Application tier
        deployer.deploy_microservices(
            infrastructure['app_tier']['services'],
            infrastructure['app_tier']['switch']
        )
        
        # 3. Data tier
        deployer.deploy_databases(
            infrastructure['data_tier']['databases'],
            infrastructure['data_tier']['switch']
        )
        
        # 4. Secure resources
        deployer.deploy_zero_trust_resources(
            infrastructure['secure_resources']['resources'],
            infrastructure['secure_resources']['switch']
        )
        
        logger.info("Infrastructure deployment completed successfully!")
        
    except Exception as e:
        logger.error(f"Deployment failed: {e}")
        raise


if __name__ == "__main__":
    main()